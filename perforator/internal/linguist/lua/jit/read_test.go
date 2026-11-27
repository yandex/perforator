package jit_test

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yandex/perforator/perforator/internal/linguist/lua/jit"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
)

type memory map[uint64][]byte

func (m memory) read(address uint64, dst []byte) error {
	for base, data := range m {
		if address >= base && address-base <= uint64(len(data)) && uint64(len(dst)) <= uint64(len(data))-(address-base) {
			copy(dst, data[address-base:])
			return nil
		}
	}
	return fmt.Errorf("unmapped remote read at %#x", address)
}

func traceMemory() (memory, jit.VM, []procfs.Mapping) {
	m := memory{}
	const g = 0x100000
	vm := jit.VM{GlobalState: g, OffsetGToVMState: 184, OffsetGToDispatch: 4000, VMStart: 0x5000, VMEnd: 0x5020}
	// Captured BC_JLOOP loads from the local Linux x64 GC64 LuaJIT binary.
	m[0x5000] = append([]byte{0x49, 0x8b, 0x8e, 0xc8, 0xf4, 0xff, 0xff, 0x48, 0x8b, 0x04, 0xc1, 0x48, 0x8b, 0x40, 0x58, 0x48, 0x83, 0xec, 0x10}, make([]byte, 13)...)
	m[g] = make([]byte, 4096)
	binary.LittleEndian.PutUint64(m[g][0x170:], 0x80000)
	binary.LittleEndian.PutUint64(m[g][192:], 0x80000)
	binary.LittleEndian.PutUint64(m[g][1128:], 0x20000)
	binary.LittleEndian.PutUint32(m[g][1140:], 2)
	m[0x80000] = make([]byte, 24)
	m[0x80000][9] = 6
	binary.LittleEndian.PutUint64(m[0x80000][16:], g)
	m[0x20000] = make([]byte, 16)
	binary.LittleEndian.PutUint64(m[0x20000][8:], 0x30000)
	m[0x30000] = make([]byte, 112)
	m[0x30000][9] = 9 // ~LJ_TTRACE.
	binary.LittleEndian.PutUint64(m[0x30000][32:], 0x30000)
	binary.LittleEndian.PutUint32(m[0x30000][84:], 11)
	binary.LittleEndian.PutUint64(m[0x30000][88:], 0x10000)
	binary.LittleEndian.PutUint16(m[0x30000][102:], 32)
	binary.LittleEndian.PutUint16(m[0x30000][104:], 1)
	m[0x70000] = make([]byte, 8)
	m[0x10000] = []byte{0x48, 0x83, 0xc4, 0xe0, 0x90, 0x48, 0x83, 0xc4, 0x20, 0xff, 0xe0}
	maps := []procfs.Mapping{{Begin: 0x10000, End: 0x11000, Permissions: procfs.MappingPermissionRXP}}
	return m, vm, maps
}

func TestReadTableFindsTraceArrayFromVMCode(t *testing.T) {
	m, vm, maps := traceMemory()
	table, err := jit.ReadTable(m.read, vm, maps)
	require.NoError(t, err)
	require.Equal(t, []uint64{0x10000, 0x10004, 0x10009}, table.StartPc)
	require.Equal(t, int32(128), table.Dict[table.Cfa[1]].GetRegisterOffset().Offset)
}

func TestSharedExitStubGetsAStopRow(t *testing.T) {
	m, vm, maps := traceMemory()
	// The conditional branch goes to a stub group in the same mcode mapping.
	m[0x10000] = []byte{0x48, 0x83, 0xc4, 0xe0, 0x0f, 0x85, 0xf6, 0x07, 0x00, 0x00, 0x90, 0x48, 0x83, 0xc4, 0x20, 0xff, 0xe0}
	binary.LittleEndian.PutUint32(m[0x30000][84:], uint32(len(m[0x10000])))
	group := make([]byte, 134)
	for i := 0; i < 32; i++ {
		group[4*i], group[4*i+1] = 0x6a, byte(i)
		if i < 31 {
			group[4*i+2], group[4*i+3] = 0xeb, byte(122-4*i)
		}
	}
	group[126], group[127], group[128] = 0x6a, 0, 0xe9
	m[0x10800] = group
	table, err := jit.ReadTable(m.read, vm, maps)
	require.NoError(t, err)
	last := len(table.StartPc) - 1
	require.Equal(t, uint64(0x10800), table.StartPc[last])
	require.Equal(t, uint64(133), table.PcRange[last])
	require.NotNil(t, table.Dict[table.Ra[last]].GetUndefined())
}

func TestChangingTraceArrayCannotPublishASnapshot(t *testing.T) {
	m, vm, maps := traceMemory()
	reads := 0
	read := func(address uint64, dst []byte) error {
		if err := m.read(address, dst); err != nil {
			return err
		}
		if address == 0x20000 {
			reads++
			if reads == 2 {
				dst[8] ^= 8
			}
		}
		return nil
	}
	_, err := jit.ReadTable(read, vm, maps)
	require.ErrorContains(t, err, "snapshot changed")
}

func exitGroup() []byte {
	group := make([]byte, 134)
	for i := 0; i < 32; i++ {
		group[4*i], group[4*i+1] = 0x6a, byte(i)
		if i < 31 {
			group[4*i+2], group[4*i+3] = 0xeb, byte(122-4*i)
		}
	}
	group[126], group[127], group[128] = 0x6a, 0, 0xe9
	return group
}

func TestReadTableDiscoversUnreferencedVMExitStubGroup(t *testing.T) {
	m, vm, maps := traceMemory()
	// Local verified x64 GC64 jit_State: exitstubgroup is trace + 1312.
	binary.LittleEndian.PutUint64(m[vm.GlobalState][1128+1312:], 0x10800)
	m[0x10800] = exitGroup()
	table, err := jit.ReadTable(m.read, vm, maps)
	require.NoError(t, err)
	last := len(table.StartPc) - 1
	require.Equal(t, uint64(0x10800), table.StartPc[last])
	require.Equal(t, uint64(133), table.PcRange[last])
	require.NotNil(t, table.Dict[table.Ra[last]].GetUndefined())
}

func TestSharedVMExitStubStopsEvenWhenCurrentGuardsAgree(t *testing.T) {
	m, vm, maps := traceMemory()
	m[0x10000] = []byte{0x48, 0x83, 0xc4, 0xe0, 0x0f, 0x85, 0xf6, 0x07, 0, 0, 0x90, 0x48, 0x83, 0xc4, 0x20, 0xff, 0xe0}
	binary.LittleEndian.PutUint32(m[0x30000][84:], uint32(len(m[0x10000])))
	binary.LittleEndian.PutUint64(m[vm.GlobalState][1128+1312:], 0x10800)
	m[0x10800] = exitGroup()
	table, err := jit.ReadTable(m.read, vm, maps)
	require.NoError(t, err)
	for _, check := range []struct{ pc uint64 }{{0x10800}, {0x10802}, {0x1087e}, {0x10880}} {
		found := false
		for i, pc := range table.StartPc {
			if check.pc >= pc && check.pc < pc+table.PcRange[i] {
				found = true
				cfa := table.Dict[table.Cfa[i]].GetRegisterOffset()
				require.Nil(t, cfa, "future traces can change CFA at the same shared stub PC %#x", check.pc)
				require.NotNil(t, table.Dict[table.Ra[i]].GetUndefined())
			}
		}
		require.True(t, found, "missing stub PC %#x", check.pc)
	}
}

func TestTraceGCAndLinkBookkeepingDoesNotInvalidateRules(t *testing.T) {
	m, vm, maps := traceMemory()
	read := func(address uint64, dst []byte) error {
		if address == 0x30000 {
			m[address][8] ^= 1  // GC mark/color.
			m[address][24] ^= 8 // GC list.
			m[address][100]++   // Child count.
			m[address][106]++   // Trace link.
		}
		return m.read(address, dst)
	}
	table, err := jit.ReadTable(read, vm, maps)
	require.NoError(t, err)
	require.Equal(t, []uint64{0x10000, 0x10004, 0x10009}, table.StartPc)
}

func TestTraceUnwindFieldsMustRemainStable(t *testing.T) {
	for _, offset := range []int{9, 32, 84, 88, 102, 104, 108} {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			m, vm, maps := traceMemory()
			reads := 0
			read := func(address uint64, dst []byte) error {
				if address == 0x30000 {
					reads++
					if reads == 2 {
						m[address][offset] ^= 1
					}
				}
				return m.read(address, dst)
			}
			_, err := jit.ReadTable(read, vm, maps)
			require.ErrorContains(t, err, "snapshot changed")
		})
	}
}

func TestCurrentLuaThreadIsNotVMLifetimeIdentity(t *testing.T) {
	m, vm, maps := traceMemory()
	// A coroutine switch can change cur_L; the main-thread anchor remains stable.
	binary.LittleEndian.PutUint64(m[vm.GlobalState][192:], 0x80000)
	m[0x80000][9] = 6
	binary.LittleEndian.PutUint64(m[vm.GlobalState][0x170:], 0)
	_, err := jit.ReadTable(m.read, vm, maps)
	require.NoError(t, err)
}

func TestVMIdentityMustRemainValidUntilPublication(t *testing.T) {
	m, vm, maps := traceMemory()
	binary.LittleEndian.PutUint64(m[vm.GlobalState][192:], 0x80000)
	m[0x80000][9] = 6
	reads := 0
	read := func(address uint64, dst []byte) error {
		if address == vm.GlobalState+192 {
			reads++
			if reads == 2 {
				binary.LittleEndian.PutUint64(m[0x80000][16:], 0)
			}
		}
		return m.read(address, dst)
	}
	_, err := jit.ReadTable(read, vm, maps)
	require.ErrorContains(t, err, "VM identity")
}

func TestTraceFreeSlotCursorDoesNotInvalidateRules(t *testing.T) {
	m, vm, maps := traceMemory()
	read := func(address uint64, dst []byte) error {
		if address == vm.GlobalState+1128 {
			m[vm.GlobalState][1128+8]++
		}
		return m.read(address, dst)
	}
	_, err := jit.ReadTable(read, vm, maps)
	require.NoError(t, err)
}
