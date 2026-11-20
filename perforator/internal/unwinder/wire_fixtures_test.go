package unwinder

import (
	"encoding/hex"
	"testing"
	"unsafe"
)

func TestWireFormat(t *testing.T) {
	// The same little-endian record and mutations are checked in raw_sample_ut.cpp.
	// Header (288 bytes: perf value triple, 5 additional_event_delta triples,
	// timedelta and 6 section descriptors), kernel/user stacks, LBR, TLS,
	// cgroups, then Python/PHP/JVM/Lua.
	// Canonical record for the wire contract in
	// agent/collector/progs/unwinder/contract/wire_types.h.
	const recordHex = "0000000001000000000000000000000000000000000000000000000000000000" +
		"01000300e80300002b0200000000000000000000000000000000000000000000" +
		"000000000000000000000000000000002a0000002b0000000000000000000000" +
		"2b020000000000002b0200000000000064000000000000000100000000000000" +
		"c800000000000000aa000000000000000100000000000000c800000000000000" +
		"bb000000000000000200000000000000c800000000000000cc00000000000000" +
		"0300000000000000c800000000000000dd000000000000000400000000000000" +
		"c800000000000000ee000000000000000500000000000000c800000000000000" +
		"c800000000000000000010001000080018003000480030017801080080017800" +
		"1000000000000000200000000000000030000000000000004000000000000000" +
		"5000000000000000070000000000000060000000000000007000000000000000" +
		"0900000000000000400000000000000001000000000000006300000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"80000000000000000200000000000000050000000000000068656c6c6f000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000002a00000000000000" +
		"200000012000000034120000000000000b000000000000003812000000000000" +
		"7856000000000000100001011000000045230000000000000c00000000000000" +
		"1000020210000000000000000000000056340000000000001800030118000000" +
		"000000000000000067450000000000000d00000000000000"
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
		{"all_sections", 792, 0, 0, 0, true},
		{"perf_padding", 796, 0, 0, 0, true},
		{"short_header", 167, 0, 0, 0, false},
		{"truncated_payload", 791, 0, 0, 0, false},
		{"wrong_tag", 792, 0, 1, 1, false},
		{"undefined_sample_type", 792, 4, 0, 1, false},
		{"unknown_sample_type", 792, 4, 6, 1, false},
		{"overlapping_sections", 792, 268, 8, 1, false},
		{"gap_between_sections", 792, 268, 24, 1, false},
		{"partial_stack", 792, 266, 15, 1, false},
		{"misaligned_section", 792, 264, 1, 1, false},
		{"section_out_of_bounds", 792, 286, 65535, 2, false},
		{"unknown_language", 792, 674, 4, 1, false},
		{"reserved_language_byte_zero", 792, 675, 0, 1, true},
		{"zero_element_size", 792, 676, 0, 1, false},
		{"misaligned_element_size", 792, 676, 7, 1, false},
		{"zero_payload_size", 792, 672, 0, 1, false},
		{"partial_language_element", 792, 672, 31, 1, false},
		{"truncated_language_header", 792, 286, 1, 1, false},
		{"language_out_of_bounds", 792, 672, 4096, 2, false},
		{"duplicate_language", 792, 714, 0, 1, false},
		{"reserved_language_byte_nonzero", 792, 675, 2, 1, true},
		// Go requires known frame layouts; C++ keeps these interpreter payloads opaque.
		{"opaque_interpreter_size", 792, 676, 16, 1, false},
		{"reserved_jvm_byte_nonzero", 792, 739, 1, 1, true},
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
				out.LuaStack.Frames[0].Value.GetLuaFrame().Linestart != 13 {
				t.Fatalf("unexpected decoded fixture: %+v", out)
			}
			// Additional event deltas: slots leader + 4 TMA metrics; only
			// the counter fields are distinct in the fixture.
			if out.AdditionalEvents[0].Counter != 0xAA || out.AdditionalEvents[1].Counter != 0xBB ||
				out.AdditionalEvents[2].Counter != 0xCC || out.AdditionalEvents[3].Counter != 0xDD ||
				out.AdditionalEvents[4].Counter != 0xEE {
				t.Fatalf("unexpected additional event deltas: %+v", out.AdditionalEvents)
			}
			for i, delta := range out.AdditionalEvents {
				if delta.Enabled != 0xc8 || delta.Running != uint64(i+1) || delta.Counter == 0 {
					t.Fatalf("unexpected additional event delta %d: %+v", i, delta)
				}
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
