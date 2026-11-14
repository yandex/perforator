package unwinder

import (
	"encoding/hex"
	"testing"
	"unsafe"
)

func TestWireFormat(t *testing.T) {
	// Cases mirror raw_sample_ut.cpp, using the current Lua frame layout here.
	// Header, kernel/user stacks, LBR, TLS, cgroups, then Python/PHP/JVM/Lua.
	const recordHex = "0000000001000000000000000000000000000000000000000000000000000000" +
		"01000300e8030000393000000000000000000000000000000000000000000000" +
		"000000000000000000000000000000002a0000002b0000000000000000000000" +
		"00000000000000002b0200000000000064000000000000000100000000000000" +
		"0100000000000000c80000000000000000001000100008001800300048003001" +
		"780108008001a000100000000000000020000000000000003000000000000000" +
		"4000000000000000500000000000000007000000000000006000000000000000" +
		"7000000000000000090000000000000040000000000000000100000000000000" +
		"6300000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000800000000000000002000000000000000500000000000000" +
		"68656c6c6f000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"2a00000000000000200000012000000034120000000000000b00000000000000" +
		"3812000000000000785600000000000010000101100000004523000000000000" +
		"0c00000000000000100002021000000000000000000000005634000000000000" +
		"4000030120000000030000000000000000100000000000000020000000000000" +
		"0000000000000000000000000000000067450000000000000d00000011000000" +
		"0500000000000000"
	const recordSize = 712
	base, err := hex.DecodeString(recordHex)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name         string
		size, offset int
		value        uint16
		width        int
		accept       bool
	}{
		{"all_sections", recordSize, 0, 0, 0, true},
		{"perf_padding", recordSize + 4, 0, 0, 0, true},
		{"short_header", 167, 0, 0, 0, false},
		{"truncated_payload", recordSize - 1, 0, 0, 0, false},
		{"wrong_tag", recordSize, 0, 1, 1, false},
		{"undefined_sample_type", recordSize, 4, 0, 1, false},
		{"unknown_sample_type", recordSize, 4, 6, 1, false},
		{"overlapping_sections", recordSize, 148, 8, 1, false},
		{"gap_between_sections", recordSize, 148, 24, 1, false},
		{"partial_stack", recordSize, 146, 15, 1, false},
		{"misaligned_section", recordSize, 144, 1, 1, false},
		{"section_out_of_bounds", recordSize, 166, 65535, 2, false},
		{"unknown_language", recordSize, 554, 4, 1, false},
		{"reserved_language_byte_zero", recordSize, 555, 0, 1, true},
		{"zero_element_size", recordSize, 556, 0, 1, false},
		{"misaligned_element_size", recordSize, 556, 7, 1, false},
		{"zero_payload_size", recordSize, 552, 0, 1, false},
		{"partial_language_element", recordSize, 552, 31, 1, false},
		{"truncated_language_header", recordSize, 166, 1, 1, false},
		{"language_out_of_bounds", recordSize, 552, 4096, 2, false},
		{"duplicate_language", recordSize, 594, 0, 1, false},
		{"reserved_language_byte_nonzero", recordSize, 555, 2, 1, true},
		// Go requires known frame layouts; C++ keeps these interpreter payloads opaque.
		{"opaque_interpreter_size", recordSize, 556, 16, 1, false},
		{"reserved_jvm_byte_nonzero", recordSize, 619, 1, 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := make([]byte, tc.size)
			copy(record, base)
			for i := len(base); i < len(record); i++ {
				record[i] = 0x5a
			}
			for i := 0; i < tc.width; i++ {
				record[tc.offset+i] = byte(tc.value >> (8 * i))
			}
			out := NewRecordSampleParsed()
			err = ParsePackedSample(record, out)
			if (err == nil) != tc.accept {
				t.Fatalf("unexpected parse result: %v", err)
			}
			if err != nil {
				return
			}
			if out.Cpu != 3 || !out.Kthread || out.Runtime != 1000 || out.Starttime != 555 || out.Pid != 42 ||
				out.KernStack[1] != 0x20 || out.UserStack[0] != 0x30 || out.Cgroups[0] != 42 ||
				out.PythonStack.Frames[0].SymbolKey.Linestart != 11 || out.PythonStack.Frames[0].InstrPtr != 0x1238 || out.PhpStack.Frames[0].SymbolKey.ObjectAddr != 0x2345 ||
				out.PhpStack.Frames[0].SymbolKey.Linestart != 12 ||
				out.JvmStack.Frames[0].MethodAddr != 0x3456 || out.LuaStack.Len != 1 ||
				out.LuaStack.Frames[0].Type != LuaFrameTypeVmInfo ||
				out.LuaStack.Frames[1].Type != LuaFrameTypeLua ||
				out.LuaStack.Frames[1].Value.GetLuaFrame().ProtoAddress != 0x4567 ||
				out.LuaStack.Frames[1].Value.GetLuaFrame().FirstLine != 13 ||
				out.LuaStack.Frames[1].Value.GetLuaFrame().CurrentLine != 17 ||
				out.LuaStack.Frames[1].Value.GetLuaFrame().BytecodeIndex != 5 {
				t.Fatalf("unexpected decoded fixture: %+v", out)
			}
			if len(out.LBR) != 2 || len(out.TLS) != 2 {
				t.Fatalf("unexpected LBR/TLS lengths: %d/%d", len(out.LBR), len(out.TLS))
			}
			if out.LBR[0].From != 0x40 || out.LBR[0].To != 0x50 || out.LBR[0].Flags != 7 ||
				out.LBR[1].From != 0x60 || out.LBR[1].To != 0x70 || out.LBR[1].Flags != 9 {
				t.Fatalf("unexpected LBR entries: %+v", out.LBR)
			}
			if out.TLS[0].Offset != 64 || out.TLS[0].Type != ThreadLocalUint64Type ||
				le.Uint64(out.TLS[0].Value.UnionBuf[:8]) != 99 ||
				out.TLS[1].Offset != 128 || out.TLS[1].Type != ThreadLocalStringType ||
				le.Uint64(out.TLS[1].Value.UnionBuf[:8]) != 5 ||
				string(out.TLS[1].Value.UnionBuf[8:13]) != "hello" {
				t.Fatalf("unexpected TLS entries: %+v", out.TLS)
			}
		})
	}
}

func TestPackedSamplePartialLbrAndTlsElements(t *testing.T) {
	for _, tc := range []struct {
		name      string
		headerOff int
		size      uint16
	}{
		{"lbr", int(unsafe.Offsetof(RecordSampleHeader{}.Lbr)), 16},
		{"tls", int(unsafe.Offsetof(RecordSampleHeader{}.Tls)), 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := buildMinimalPackedSample()
			// putSectionDesc keeps subsequent sections contiguous. Both sizes
			// are 8-byte aligned but not multiples of their element sizes.
			putSectionDesc(record, tc.headerOff, 0, tc.size)
			record = append(record, make([]byte, tc.size)...)
			if err := ParsePackedSample(record, NewRecordSampleParsed()); err == nil {
				t.Fatal("accepted a partial section element")
			}
		})
	}
}

func TestPackedSampleCapacityAndPadding(t *testing.T) {
	for _, fixedSize := range []bool{false, true} {
		data := buildMinimalPackedSample()
		size := uint16(8)
		if !fixedSize {
			size = maxPackedDataSize
		}
		putSectionDesc(data, sdKernStack, 0, size)
		data = append(data, make([]byte, maxPackedDataSize)...)
		for _, padding := range []int{0, 4} {
			record := append(append([]byte(nil), data...), []byte{0x5a, 0x5a, 0x5a, 0x5a}[:padding]...)
			out := NewRecordSampleParsed()
			if err := ParsePackedSample(record, out); err != nil {
				t.Fatal(err)
			}
			if len(out.KernStack) != int(size)/8 {
				t.Fatal("wrong stack length")
			}
		}
		putSectionDesc(data, sdLangSect, maxPackedDataSize, 4)
		if ParsePackedSample(append(data, 1, 2, 3, 4), NewRecordSampleParsed()) == nil {
			t.Fatal("accepted padding as payload")
		}
	}
	data := append(buildMinimalPackedSample(), make([]byte, maxPackedDataSize+5)...)
	if ParsePackedSample(data, NewRecordSampleParsed()) == nil {
		t.Fatal("accepted oversized record")
	}
}
