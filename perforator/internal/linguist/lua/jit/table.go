package jit

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	"golang.org/x/arch/x86/x86asm"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/unwind"
)

const cframeSizeJIT = 96

// Trace describes the native frame at a LuaJIT trace entry on Linux x64 GC64.
type Trace struct {
	Address          uint64
	Code             []byte
	StackAdjustment  uint16
	ParentAdjustment uint16
}

type nativeRow struct {
	begin, end uint64
	offset     int32 // A negative offset stops unwinding.
}

// BuildTable describes each reachable instruction using the existing DWARF rules.
// Callers must verify the 96-byte JIT C frame layout before supplying traces.
func BuildTable(traces []Trace) (*unwind.UnwindTable, error) {
	var rows []nativeRow
	for _, trace := range traces {
		if trace.Address == 0 || len(trace.Code) == 0 || len(trace.Code) > 1<<20 || trace.Address > math.MaxUint64-uint64(len(trace.Code)) {
			return nil, fmt.Errorf("invalid trace interval at %#x", trace.Address)
		}
		if trace.ParentAdjustment > trace.StackAdjustment {
			return nil, fmt.Errorf("trace stack shrinks below its parent")
		}
		decoded, err := traceRows(trace)
		if err != nil {
			return nil, fmt.Errorf("trace at %#x: %w", trace.Address, err)
		}
		rows = append(rows, decoded...)
	}
	return rowsToTable(rows)
}

func traceRows(trace Trace) ([]nativeRow, error) {
	offsets := map[int]int32{0: cframeSizeJIT + int32(trace.ParentAdjustment)}
	lengths := make(map[int]int)
	owners := make([]int, len(trace.Code))
	for i := range owners {
		owners[i] = -1
	}
	queue := []int{0}
	reachedBody := false
	for len(queue) > 0 {
		pc := queue[0]
		queue = queue[1:]
		if owners[pc] >= 0 && owners[pc] != pc {
			return nil, fmt.Errorf("branch into an instruction at %#x", pc)
		}
		inst, err := decode(trace.Code[pc:])
		if err != nil {
			return nil, fmt.Errorf("instruction at +%#x: %w", pc, err)
		}
		for i := pc; i < pc+inst.Len; i++ {
			if owners[i] >= 0 && owners[i] != pc {
				return nil, fmt.Errorf("overlapping instructions")
			}
			owners[i] = pc
		}
		lengths[pc] = inst.Len
		offset := offsets[pc]
		reachedBody = reachedBody || offset == cframeSizeJIT+int32(trace.StackAdjustment)
		nextOffset, err := stackOffset(inst, offset)
		if err != nil {
			return nil, err
		}
		if nextOffset < cframeSizeJIT || nextOffset > cframeSizeJIT+int32(trace.StackAdjustment) {
			return nil, fmt.Errorf("stack adjustment disagrees with trace metadata at %#x", pc)
		}
		successors := []int{pc + inst.Len}
		if inst.Op == x86asm.JMP || strings.HasPrefix(inst.Op.String(), "J") || strings.HasPrefix(inst.Op.String(), "LOOP") {
			rel, direct := inst.Args[0].(x86asm.Rel)
			if inst.Op == x86asm.JMP {
				successors = nil
			}
			if direct {
				successors = append(successors, pc+inst.Len+int(rel))
			}
		}
		for _, next := range successors {
			if next < 0 || next >= len(trace.Code) {
				continue
			}
			if previous, ok := offsets[next]; ok {
				if previous != nextOffset {
					return nil, fmt.Errorf("conflicting stack offsets at %#x", next)
				}
				continue
			}
			offsets[next] = nextOffset
			queue = append(queue, next)
		}
	}
	if !reachedBody {
		return nil, fmt.Errorf("body stack adjustment is absent")
	}
	var rows []nativeRow
	for pc, length := range lengths {
		rows = append(rows, nativeRow{trace.Address + uint64(pc), trace.Address + uint64(pc+length), offsets[pc]})
	}
	slices.SortFunc(rows, func(a, b nativeRow) int { return cmp.Compare(a.begin, b.begin) })
	// Unreachable bytes get a stop row, so a lookup cannot fall back to a frame pointer.
	var result []nativeRow
	end := trace.Address
	for _, row := range rows {
		if end < row.begin {
			result = append(result, nativeRow{end, row.begin, -1})
		}
		result = append(result, row)
		end = row.end
	}
	if end < trace.Address+uint64(len(trace.Code)) {
		result = append(result, nativeRow{end, trace.Address + uint64(len(trace.Code)), -1})
	}
	return result, nil
}

func decode(code []byte) (x86asm.Inst, error) {
	// Older x86 decoders expose ENDBR64 as an unknown opcode.
	if len(code) >= 4 && code[0] == 0xf3 && code[1] == 0x0f && code[2] == 0x1e && code[3] == 0xfa {
		return x86asm.Inst{Op: x86asm.NOP, Len: 4}, nil
	}
	if len(code) >= 5 && code[0] == 0xc4 {
		rorx := code[1]&0x1f == 3 && code[2]&0x7f == 0x7b && code[3] == 0xf0
		shift := code[1]&0x1f == 2 && code[2]&4 == 0 && code[2]&3 != 0 && code[3] == 0xf7
		if rorx || shift {
			// x86asm lacks these BMI2 instructions. MOV has the same ModRM/SIB
			// operands and destination, which determines their stack effect.
			rex := byte(0x40) | (code[2]>>4)&8 | (^code[1]>>5)&7
			operands := append([]byte{rex, 0x8b}, code[4:min(len(code), 15)]...)
			inst, err := x86asm.Decode(operands, 64)
			inst.Len += 2 // Replace the one-byte REX with the three-byte VEX.
			if rorx {
				inst.Len++ // Rotation immediate.
			}
			if err != nil || inst.Op != x86asm.MOV || inst.Len > len(code) {
				return inst, fmt.Errorf("invalid BMI2 instruction")
			}
			return inst, nil
		}
	}
	inst, err := x86asm.Decode(code, 64)
	if err != nil || inst.Op == 0 || inst.Len == 0 {
		return inst, fmt.Errorf("invalid x64 instruction")
	}
	return inst, nil
}

func stackOffset(inst x86asm.Inst, offset int32) (int32, error) {
	switch inst.Op {
	case x86asm.PUSH, x86asm.POP, x86asm.PUSHF, x86asm.POPF, x86asm.PUSHFQ, x86asm.POPFQ, x86asm.ENTER, x86asm.LEAVE, x86asm.RET, x86asm.LRET, x86asm.IRET, x86asm.IRETQ:
		return 0, fmt.Errorf("unsupported implicit stack write: %s", inst.Op)
	}
	if inst.Op == x86asm.XCHG && isSP(inst.Args[1]) {
		return 0, fmt.Errorf("unsupported exchange with stack pointer")
	}
	if !isSP(inst.Args[0]) || inst.Op == x86asm.CMP || inst.Op == x86asm.TEST || inst.Op == x86asm.BT {
		return offset, nil
	}
	imm, ok := inst.Args[1].(x86asm.Imm)
	if inst.Args[0] != x86asm.RSP || !ok || (inst.Op != x86asm.ADD && inst.Op != x86asm.SUB) {
		return 0, fmt.Errorf("unsupported stack write: %s", inst.Op)
	}
	delta := int64(imm)
	if inst.Op == x86asm.SUB {
		delta = -delta
	}
	value := int64(offset) - delta
	if value < 0 || value > math.MaxInt32 {
		return 0, fmt.Errorf("stack offset overflow")
	}
	return int32(value), nil
}

func isSP(arg x86asm.Arg) bool {
	return arg == x86asm.RSP || arg == x86asm.ESP || arg == x86asm.SP || arg == x86asm.SPB
}

func rowsToTable(rows []nativeRow) (*unwind.UnwindTable, error) {
	slices.SortFunc(rows, func(a, b nativeRow) int { return cmp.Compare(a.begin, b.begin) })
	table := &unwind.UnwindTable{}
	for _, row := range rows {
		n := len(table.StartPc)
		if n > 0 {
			previousEnd := table.StartPc[n-1] + table.PcRange[n-1]
			if row.offset < 0 && row.begin == table.StartPc[n-1] && row.end == previousEnd && table.Dict[table.Ra[n-1]].GetUndefined() != nil {
				continue
			}
			if row.begin < previousEnd {
				return nil, fmt.Errorf("overlapping trace code at %#x", row.begin)
			}
			previous := table.Dict[table.Cfa[n-1]].GetRegisterOffset()
			if row.begin == previousEnd && ((row.offset < 0 && previous == nil) || (previous != nil && previous.Offset == row.offset)) {
				table.PcRange[n-1] += row.end - row.begin
				continue
			}
		}
		base := uint32(len(table.Dict))
		cfa := &unwind.UnwindRule{Kind: &unwind.UnwindRule_RegisterOffset{RegisterOffset: &unwind.UnwindRule_RegisterPlusOffset{Register: 7, Offset: row.offset}}}
		ra := &unwind.UnwindRule{Kind: &unwind.UnwindRule_CfaMinus8{CfaMinus8: &unwind.UnwindRule_CFAMinus8{}}}
		fp := &unwind.UnwindRule{Kind: &unwind.UnwindRule_CfaPlusOffset{CfaPlusOffset: &unwind.UnwindRule_CFAPlusOffset{Offset: -16}}}
		if row.offset < 0 {
			ra = &unwind.UnwindRule{Kind: &unwind.UnwindRule_Undefined_{Undefined: &unwind.UnwindRule_Undefined{}}}
			cfa = &unwind.UnwindRule{}
		}
		table.Dict = append(table.Dict, cfa, fp, ra)
		table.StartPc = append(table.StartPc, row.begin)
		table.PcRange = append(table.PcRange, row.end-row.begin)
		table.Cfa = append(table.Cfa, base)
		table.Rbp = append(table.Rbp, base+1)
		table.Ra = append(table.Ra, base+2)
	}
	return table, nil
}
