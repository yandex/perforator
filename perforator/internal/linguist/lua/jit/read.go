package jit

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"golang.org/x/arch/x86/x86asm"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/unwind"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
)

type Reader func(address uint64, dst []byte) error

type VM struct {
	GlobalState       uint64
	OffsetGToVMState  uint64
	OffsetGToDispatch uint64
	VMStart           uint64
	VMEnd             uint64
}

// ReadTable takes a bounded, revalidated snapshot of the Linux x64 GC64 trace set.
// Offsets in the GCtrace prefix are checked against the VM's trace-entry code.
func ReadTable(read Reader, vm VM, mappings []procfs.Mapping) (*unwind.UnwindTable, error) {
	return readTable(read, vm, mappings, &vmLayout{})
}

// vmLayout contains field addresses and code layout only. Mutable field values
// (including the trace-array pointer) are read and revalidated on every refresh.
type vmLayout struct {
	traceArrayAddress    uint64
	exitStubArrayAddress uint64
	mainThread           uint64
}

// Only fields that identify the trace and determine its unwind rules belong in
// the snapshot. GC colors/lists, link bookkeeping, and child counts can change
// while a compiled trace's native frame layout remains unchanged.
type traceState struct {
	kind                     byte
	ir, mcode                uint64
	size                     uint32
	adjustment, number, root uint16
}

func readTraceState(read Reader, address uint64) (traceState, error) {
	data, err := readBytes(read, address, 112)
	if err != nil {
		return traceState{}, err
	}
	return traceState{kind: data[9], ir: binary.LittleEndian.Uint64(data[32:]), size: binary.LittleEndian.Uint32(data[84:]), mcode: binary.LittleEndian.Uint64(data[88:]), adjustment: binary.LittleEndian.Uint16(data[102:]), number: binary.LittleEndian.Uint16(data[104:]), root: binary.LittleEndian.Uint16(data[108:])}, nil
}

func readTable(read Reader, vm VM, mappings []procfs.Mapping, layout *vmLayout) (*unwind.UnwindTable, error) {
	if vm.GlobalState == 0 || vm.GlobalState >= 1<<47 || vm.VMEnd <= vm.VMStart || vm.VMEnd-vm.VMStart > 1<<16 || vm.OffsetGToVMState == 0 || vm.OffsetGToVMState > 1<<16 || vm.OffsetGToDispatch > 1<<16 {
		return nil, fmt.Errorf("invalid VM layout")
	}
	main, err := readVMIdentity(read, vm, mappings, layout.mainThread)
	if err != nil {
		return nil, err
	}
	layout.mainThread = main

	if layout.traceArrayAddress == 0 {
		code, err := readBytes(read, vm.VMStart, int(vm.VMEnd-vm.VMStart))
		if err != nil {
			return nil, err
		}
		displacement, err := traceArrayDisplacement(code)
		if err != nil {
			return nil, err
		}
		address := int64(vm.GlobalState) + int64(vm.OffsetGToDispatch) + displacement
		if address <= 0 {
			return nil, fmt.Errorf("invalid trace array address")
		}
		layout.traceArrayAddress = uint64(address)
		// Verified Linux x64 GC64 LuaJIT 2.1 layout: J.trace + 1312.
		// Every non-null pointer and group number is checked against stub code.
		layout.exitStubArrayAddress = uint64(address) + 1312
	}
	header, err := readBytes(read, layout.traceArrayAddress, 16)
	if err != nil {
		return nil, err
	}

	array := binary.LittleEndian.Uint64(header)
	count := binary.LittleEndian.Uint32(header[12:])
	if count > 65536 || (count != 0 && array == 0) {
		return nil, fmt.Errorf("invalid trace array size %d", count)
	}
	slots, err := readBytes(read, array, int(count)*8)
	if err != nil {
		return nil, err
	}
	type object struct {
		address uint64
		state   traceState
		code    []byte
		parent  uint16
	}
	objects := make(map[uint16]object)
	budget := 8 << 20
	for slot := uint32(1); slot < count; slot++ {
		address := binary.LittleEndian.Uint64(slots[slot*8:])
		if address == 0 {
			continue
		}
		state, err := readTraceState(read, address)
		if err != nil {
			return nil, err
		}
		if state.kind != 9 || state.number != uint16(slot) {
			return nil, fmt.Errorf("trace slot %d changed", slot)
		}
		mcode := state.mcode
		size := state.size
		if size == 0 || size > 1<<20 || int(size) > budget || !inJITMapping(mappings, mcode, uint64(size)) {
			return nil, fmt.Errorf("invalid code range for trace %d", slot)
		}
		budget -= int(size)
		traceCode, err := readBytes(read, mcode, int(size))
		if err != nil {
			return nil, err
		}
		base, err := readBytes(read, state.ir+0x8000*8, 8)
		if err != nil {
			return nil, err
		}
		parent := binary.LittleEndian.Uint16(base)
		if (state.root == 0) != (parent == 0) {
			return nil, fmt.Errorf("invalid parent of trace %d", slot)
		}
		objects[uint16(slot)] = object{address, state, traceCode, parent}
	}
	var rows []nativeRow
	stubGroups := make(map[uint64]bool)
	type stubSnapshot struct {
		end  uint64
		code []byte
	}
	vmStubGroups := make(map[uint64]stubSnapshot)
	stubPointers, err := readBytes(read, layout.exitStubArrayAddress, 16*8)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 16; i++ {
		address := binary.LittleEndian.Uint64(stubPointers[i*8:])
		if address == 0 {
			continue
		}
		group, end, ok := exitStubGroup(read, mappings, address)
		if !ok || group != address {
			return nil, fmt.Errorf("invalid VM exit stub group %d", i)
		}
		code, err := readBytes(read, address, 128)
		if err != nil || code[1] != byte(i*32) || code[127] != byte(i/8) {
			return nil, fmt.Errorf("invalid VM exit stub group number %d", i)
		}
		stubGroups[group] = true
		groupCode, err := readBytes(read, group, int(end-group))
		if err != nil {
			return nil, err
		}
		vmStubGroups[group] = stubSnapshot{end, groupCode}
	}
	for _, obj := range objects {
		adjustment := obj.state.adjustment
		var parentAdjustment uint16
		if obj.parent != 0 {
			parent, ok := objects[obj.parent]
			if !ok {
				return nil, fmt.Errorf("missing parent trace %d", obj.parent)
			}
			parentAdjustment = parent.state.adjustment
		}
		address := obj.state.mcode
		trace := Trace{address, obj.code, adjustment, parentAdjustment}
		decoded, err := traceRows(trace)
		if err != nil || parentAdjustment > adjustment {
			rows = append(rows, nativeRow{address, address + uint64(len(obj.code)), -1})
		} else {
			rows = append(rows, decoded...)
		}
		// Shared exit stubs have trace-dependent layouts. Publish stop rows
		// across verified stub groups so the native unwinder cannot guess a FP.
		for pc := 0; pc < len(obj.code); {
			inst, err := decode(obj.code[pc:])
			if err != nil {
				break
			}
			if rel, ok := inst.Args[0].(x86asm.Rel); ok && (inst.Op == x86asm.JMP || isConditionalBranch(inst)) {
				target := uint64(int64(address) + int64(pc+inst.Len) + int64(rel))
				if target < address || target >= address+uint64(len(obj.code)) {
					if group, end, ok := exitStubGroup(read, mappings, target); ok && !stubGroups[group] {
						stubGroups[group] = true
						rows = append(rows, nativeRow{group, end, -1})
					}
				}
			}
			pc += inst.Len
		}
		// Trace objects and exit branches can be modified during compilation.
		state, err := readTraceState(read, obj.address)
		if err != nil {
			return nil, err
		}
		if state != obj.state {
			return nil, fmt.Errorf("LuaJIT snapshot changed: trace fields at %#x", obj.address)
		}
		base, err := readBytes(read, state.ir+0x8000*8, 2)
		if err != nil {
			return nil, err
		}
		if parent := binary.LittleEndian.Uint16(base); parent != obj.parent {
			return nil, fmt.Errorf("LuaJIT snapshot changed: trace parent at %#x", obj.address)
		}
		if err := checkBytes(read, address, obj.code, "trace code"); err != nil {
			return nil, err
		}
	}
	for group, snapshot := range vmStubGroups {
		if err := checkBytes(read, group, snapshot.code, "exit stub code"); err != nil {
			return nil, err
		}
		// Future traces can introduce a different CFA at these same PCs.
		// A snapshot-only agreement cannot establish a safe static unwind rule.
		rows = append(rows, nativeRow{group, snapshot.end, -1})
	}
	if err := checkBytes(read, array, slots, "trace slots"); err != nil {
		return nil, err
	}
	latest, err := readBytes(read, layout.traceArrayAddress, 16)
	if err != nil {
		return nil, err
	}
	if pointer := binary.LittleEndian.Uint64(latest); pointer != array {
		return nil, fmt.Errorf("LuaJIT snapshot changed: trace array pointer")
	}
	if size := binary.LittleEndian.Uint32(latest[12:]); size != count {
		return nil, fmt.Errorf("LuaJIT snapshot changed: trace array size")
	}
	if err := checkBytes(read, layout.exitStubArrayAddress, stubPointers, "exit stub pointers"); err != nil {
		return nil, err
	}
	if _, err := readVMIdentity(read, vm, mappings, layout.mainThread); err != nil {
		return nil, err
	}
	return rowsToTable(rows)
}

type vmIdentityError struct{ reason string }

func (e *vmIdentityError) Error() string { return "VM identity invalid: " + e.reason }

// In GC64 global_State, mainthref follows the 32-bit vmstate field with
// pointer alignment. Unlike cur_L, this anchor does not change on thread switches.
func readVMIdentity(read Reader, vm VM, mappings []procfs.Mapping, expected uint64) (uint64, error) {
	address := vm.GlobalState + ((vm.OffsetGToVMState + 4 + 7) &^ uint64(7))
	data, err := readBytes(read, address, 8)
	if err != nil {
		return 0, vmIdentityReadError(mappings, address, err)
	}
	main := binary.LittleEndian.Uint64(data)
	if main == 0 || main >= 1<<47 || main&7 != 0 {
		return 0, &vmIdentityError{fmt.Sprintf("invalid mainthref=%#x at %#x", main, address)}
	}
	if expected != 0 && main != expected {
		return 0, &vmIdentityError{fmt.Sprintf("mainthref at %#x changed from %#x to %#x", address, expected, main)}
	}
	thread, err := readBytes(read, main, 24)
	if err != nil {
		return 0, vmIdentityReadError(mappings, main, err)
	}
	global := binary.LittleEndian.Uint64(thread[16:])
	if thread[9] != 6 || global != vm.GlobalState {
		return 0, &vmIdentityError{fmt.Sprintf("main thread %#x: gct=%d glref=%#x expected gct=6 glref=%#x", main, thread[9], global, vm.GlobalState)}
	}
	return main, nil
}

func vmIdentityReadError(mappings []procfs.Mapping, address uint64, err error) error {
	for _, m := range mappings {
		if address >= m.Begin && address < m.End {
			return fmt.Errorf("VM identity read at %#x: %w", address, err)
		}
	}
	return &vmIdentityError{fmt.Sprintf("identity address %#x is unmapped: %v", address, err)}
}

func readBytes(read Reader, address uint64, size int) ([]byte, error) {
	if size == 0 {
		return nil, nil
	}
	if address == 0 || address >= 1<<47 || size < 0 || uint64(size) > 1<<47-address {
		return nil, fmt.Errorf("invalid remote address %#x", address)
	}
	data := make([]byte, size)
	if err := read(address, data); err != nil {
		return nil, err
	}
	return data, nil
}

func checkBytes(read Reader, address uint64, want []byte, stage string) error {
	got, err := readBytes(read, address, len(want))
	if err != nil {
		return err
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("LuaJIT snapshot changed: %s at %#x", stage, address)
	}
	return nil
}

func traceArrayDisplacement(code []byte) (int64, error) {
	var result int64
	for pc := 0; pc < len(code); pc++ {
		first, err := decode(code[pc:])
		if err != nil || first.Op != x86asm.MOV {
			continue
		}
		table, ok := first.Args[1].(x86asm.Mem)
		reg, regOK := first.Args[0].(x86asm.Reg)
		if !ok || !regOK || table.Base != x86asm.R14 || table.Index != 0 || int32(table.Disp) >= 0 || first.DataSize != 64 {
			continue
		}
		next := pc + first.Len
		if next >= len(code) {
			continue
		}
		second, err := decode(code[next:])
		if err != nil || second.Op != x86asm.MOV || second.DataSize != 64 {
			continue
		}
		slot, ok := second.Args[1].(x86asm.Mem)
		traceReg, regOK := second.Args[0].(x86asm.Reg)
		if !ok || !regOK || slot.Base != reg || slot.Scale != 8 || slot.Index == 0 || slot.Disp != 0 {
			continue
		}
		next += second.Len
		if next >= len(code) {
			continue
		}
		third, err := decode(code[next:])
		if err != nil || third.Op != x86asm.MOV || third.DataSize != 64 {
			continue
		}
		mcode, ok := third.Args[1].(x86asm.Mem)
		if !ok || mcode.Base != traceReg || mcode.Index != 0 || mcode.Disp != 88 {
			continue
		}
		if result != 0 && result != int64(int32(table.Disp)) {
			return 0, fmt.Errorf("ambiguous trace table layout")
		}
		result = int64(int32(table.Disp))
	}
	if result == 0 {
		return 0, fmt.Errorf("GC64 trace entry sequence not found")
	}
	return result, nil
}

func inJITMapping(mappings []procfs.Mapping, address, size uint64) bool {
	for _, m := range mappings {
		if m.Path == "" && m.Permissions&procfs.MappingPermissionExecutable != 0 && address >= m.Begin && address < m.End && size <= m.End-address {
			return true
		}
	}
	return false
}

func exitStubGroup(read Reader, mappings []procfs.Mapping, target uint64) (uint64, uint64, bool) {
	if !inJITMapping(mappings, target, 2) {
		return 0, 0, false
	}
	push, err := readBytes(read, target, 2)
	if err != nil || push[0] != 0x6a {
		return 0, 0, false
	}
	group := target - uint64(push[1]%32)*4
	const size = 134
	if !inJITMapping(mappings, group, size) {
		return 0, 0, false
	}
	code, err := readBytes(read, group, size)
	if err != nil {
		return 0, 0, false
	}
	for i := 0; i < 32; i++ {
		if code[4*i] != 0x6a || code[4*i+1] != byte(int(push[1])/32*32+i) {
			return 0, 0, false
		}
		if i < 31 && (code[4*i+2] != 0xeb || code[4*i+3] != byte(122-4*i)) {
			return 0, 0, false
		}
	}
	if code[126] != 0x6a {
		return 0, 0, false
	}
	if code[128] == 0xe9 {
		return group, group + 133, true
	}
	if code[128] == 0xff && code[129] == 0x25 {
		return group, group + 134, true
	}
	return 0, 0, false
}

func isConditionalBranch(inst x86asm.Inst) bool {
	name := inst.Op.String()
	return len(name) > 1 && name[0] == 'J' && inst.Op != x86asm.JMP
}
