package unwinder

import (
	"encoding/hex"
	"testing"
	"unsafe"
)

func TestWireFormat(t *testing.T) {
	// The same little-endian record and mutations are checked in raw_sample_ut.cpp.
	// Header, kernel/user stacks, LBR, TLS, cgroups, then Python/PHP/JVM/Lua.
	const recordHex = "0000000001000000000000000000000000000000000000000000000000000000" +
		"01000300e8030000393000000000000000000000000000000000000000000000" +
		"000000000000000000000000000000002a0000002b0000000000000000000000" +
		"00000000000000002b020000000000006400000000000000c800000000000000" +
		"0000100010000800180030004800300178010800800178001000000000000000" +
		"2000000000000000300000000000000040000000000000005000000000000000" +
		"0700000000000000600000000000000070000000000000000900000000000000" +
		"4000000000000000010000000000000063000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000008000000000000000" +
		"0200000000000000050000000000000068656c6c6f0000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"000000000000000000000000000000002a000000000000002000000120000000" +
		"34120000000000002a0000000b00000038120000000000007856000000000000" +
		"100001011000000045230000000000002a0000000c0000001000020210000000" +
		"0000000000000000563400000000000018000301180000000000000000000000" +
		"67450000000000002a0000000d000000"
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
		{"all_sections", 656, 0, 0, 0, true},
		{"perf_padding", 660, 0, 0, 0, true},
		{"short_header", 151, 0, 0, 0, false},
		{"truncated_payload", 655, 0, 0, 0, false},
		{"wrong_tag", 656, 0, 1, 1, false},
		{"undefined_sample_type", 656, 4, 0, 1, false},
		{"unknown_sample_type", 656, 4, 6, 1, false},
		{"overlapping_sections", 656, 132, 8, 1, false},
		{"gap_between_sections", 656, 132, 24, 1, false},
		{"partial_stack", 656, 130, 15, 1, false},
		{"misaligned_section", 656, 128, 1, 1, false},
		{"section_out_of_bounds", 656, 150, 65535, 2, false},
		{"unknown_language", 656, 538, 4, 1, false},
		{"unknown_payload_kind", 656, 539, 0, 1, false},
		{"zero_element_size", 656, 540, 0, 1, false},
		{"misaligned_element_size", 656, 540, 7, 1, false},
		{"zero_payload_size", 656, 536, 0, 1, false},
		{"partial_language_element", 656, 536, 31, 1, false},
		{"truncated_language_header", 656, 150, 1, 1, false},
		{"language_out_of_bounds", 656, 536, 4096, 2, false},
		{"duplicate_language", 656, 578, 0, 1, false},
		{"non_jvm_annotations", 656, 539, 2, 1, false},
		// Go requires known frame layouts; C++ keeps these interpreter payloads opaque.
		{"opaque_interpreter_size", 656, 540, 16, 1, false},
		{"jvm_interpreter_payload", 656, 603, 1, 1, false},
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
			if out.Cpu != 3 || !out.Kthread || out.Runtime != 1000 || out.Starttime != 555 ||
				out.KernStack[1] != 0x20 || out.UserStack[0] != 0x30 || out.Cgroups[0] != 42 ||
				out.PythonStack.Frames[0].InstrPtr != 0x1238 || out.PhpStack.Frames[0].SymbolKey.ObjectAddr != 0x2345 ||
				out.JvmStack.Frames[0].MethodAddr != 0x3456 || out.LuaStack.Len != 1 {
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
