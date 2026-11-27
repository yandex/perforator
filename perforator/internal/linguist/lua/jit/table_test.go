package jit_test

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/yandex/perforator/perforator/internal/linguist/lua/jit"
	"testing"
)

// Captured from Linux x64 GC64 LuaJIT with root, side, and FFI traces.
//
//go:embed testdata/traces.json
var capturedTraces []byte

func TestCapturedLuaJITTracesRestoreNativeCallers(t *testing.T) {
	var traces []struct {
		jit.Trace
		Hex string
	}
	require.NoError(t, json.Unmarshal(capturedTraces, &traces))
	for i, trace := range traces {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var err error
			trace.Code, err = hex.DecodeString(trace.Hex)
			require.NoError(t, err)
			table, err := jit.BuildTable([]jit.Trace{trace.Trace})
			require.NoError(t, err)
			require.NotEmpty(t, table.StartPc)
		})
	}
}

func TestBMI2ShiftCannotWriteStackPointer(t *testing.T) {
	// shlx rsp, rax, rax must fail the same stack-write validation as MOV rsp.
	_, err := jit.BuildTable([]jit.Trace{{Address: 0x6000, Code: []byte{0xc4, 0xe2, 0xf9, 0xf7, 0xe0, 0xff, 0xe0}}})
	require.ErrorContains(t, err, "unsupported stack write")
}

func TestLargeTraceFrameUsesSignedImmediate(t *testing.T) {
	// add rsp,-256; nop; add rsp,256; jmp rax.
	table, err := jit.BuildTable([]jit.Trace{{
		Address: 0x7000, StackAdjustment: 256,
		Code: []byte{0x48, 0x81, 0xc4, 0x00, 0xff, 0xff, 0xff, 0x90, 0x48, 0x81, 0xc4, 0x00, 0x01, 0x00, 0x00, 0xff, 0xe0},
	}})
	require.NoError(t, err)
	require.Equal(t, []uint64{0x7000, 0x7007, 0x700f}, table.StartPc)
	require.Equal(t, int32(352), table.Dict[table.Cfa[1]].GetRegisterOffset().Offset)
}

func TestRootTraceRestoresNativeCaller(t *testing.T) {
	// add rsp,-32; nop; add rsp,32; jmp rax.
	table, err := jit.BuildTable([]jit.Trace{{
		Address:         0x1000,
		Code:            []byte{0x48, 0x83, 0xc4, 0xe0, 0x90, 0x48, 0x83, 0xc4, 0x20, 0xff, 0xe0},
		StackAdjustment: 32,
	}})
	require.NoError(t, err)
	require.Equal(t, []uint64{0x1000, 0x1004, 0x1009}, table.StartPc)
	require.Equal(t, []uint64{4, 5, 2}, table.PcRange)
	for index, offset := range []int32{96, 128, 96} {
		require.Equal(t, offset, table.Dict[table.Cfa[index]].GetRegisterOffset().Offset)
		require.Equal(t, uint32(7), table.Dict[table.Cfa[index]].GetRegisterOffset().Register)
		require.Equal(t, int32(-16), table.Dict[table.Rbp[index]].GetCfaPlusOffset().Offset)
		require.NotNil(t, table.Dict[table.Ra[index]].GetCfaMinus8())
	}
}

func TestSideTraceUsesImmediateParentsFrame(t *testing.T) {
	// A guard before growth can leave with the parent's stack. Calls return
	// without changing the offset, and the tail restores the whole child frame.
	table, err := jit.BuildTable([]jit.Trace{{
		Address:         0x2000,
		Code:            []byte{0xf3, 0x0f, 0x1e, 0xfa, 0x75, 0x7f, 0x48, 0x83, 0xc4, 0xe0, 0xe8, 0x70, 0, 0, 0, 0x48, 0x83, 0xc4, 0x40, 0xff, 0xe0},
		StackAdjustment: 64, ParentAdjustment: 32,
	}})
	require.NoError(t, err)
	require.Equal(t, []uint64{0x2000, 0x200a, 0x2013}, table.StartPc)
	require.Equal(t, []uint64{10, 9, 2}, table.PcRange)
	require.Equal(t, int32(128), table.Dict[table.Cfa[0]].GetRegisterOffset().Offset)
	require.Equal(t, int32(160), table.Dict[table.Cfa[1]].GetRegisterOffset().Offset)
	require.Equal(t, int32(96), table.Dict[table.Cfa[2]].GetRegisterOffset().Offset)
}

func TestConflictingLoopOffsetsAreRejected(t *testing.T) {
	_, err := jit.BuildTable([]jit.Trace{{Address: 0x3000, StackAdjustment: 32, Code: []byte{0x48, 0x83, 0xc4, 0xe0, 0xeb, 0xfa}}})
	require.ErrorContains(t, err, "conflicting stack offsets")
}

func TestTraceMetadataMustMatchBodyAllocation(t *testing.T) {
	_, err := jit.BuildTable([]jit.Trace{{Address: 0x4000, StackAdjustment: 32, Code: []byte{0x90, 0xff, 0xe0}}})
	require.ErrorContains(t, err, "body stack adjustment")
}
