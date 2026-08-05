package unwinder

import (
	"encoding/binary"
	"testing"
	"unsafe"
)

var (
	sdKernStack = int(unsafe.Offsetof(RecordSampleHeader{}.KernStack))
	sdUserStack = int(unsafe.Offsetof(RecordSampleHeader{}.UserStack))
	sdCgroups   = int(unsafe.Offsetof(RecordSampleHeader{}.Cgroups))
	sdLangSect  = int(unsafe.Offsetof(RecordSampleHeader{}.LanguageSections))
)

func buildMinimalPackedSample() []byte {
	buf := make([]byte, recordSampleHeaderSize)
	le := binary.LittleEndian

	buf[0] = 0                      // tag
	le.PutUint32(buf[4:8], 1)       // sample_type
	le.PutUint16(buf[34:36], 3)     // cpu
	le.PutUint32(buf[36:40], 1000)  // runtime
	le.PutUint64(buf[40:48], 12345) // collection_time
	le.PutUint32(buf[80:84], 42)    // pid
	le.PutUint32(buf[84:88], 43)    // tid
	le.PutUint64(buf[96:104], 99)   // parent_cgroup
	le.PutUint64(buf[104:112], 555) // starttime
	le.PutUint64(buf[112:120], 100) // value
	le.PutUint64(buf[120:128], 200) // timedelta
	// All section descriptors are zero (offset=0, size=0)

	return buf
}

// putSectionDesc writes a section_desc at the given header offset.
func putSectionDesc(buf []byte, headerOff int, offset, size uint16) {
	le := binary.LittleEndian
	le.PutUint16(buf[headerOff:], offset)
	le.PutUint16(buf[headerOff+2:], size)
}

func TestParsePackedSampleMinimal(t *testing.T) {
	data := buildMinimalPackedSample()
	out := NewRecordSampleParsed()

	if err := ParsePackedSample(data, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Pid != 42 {
		t.Errorf("Pid = %d, want 42", out.Pid)
	}
	if out.Tid != 43 {
		t.Errorf("Tid = %d, want 43", out.Tid)
	}
	if out.Cpu != 3 {
		t.Errorf("Cpu = %d, want 3", out.Cpu)
	}
	if out.Value != 100 {
		t.Errorf("Value = %d, want 100", out.Value)
	}
	if len(out.KernStack) != 0 || len(out.UserStack) != 0 || len(out.Cgroups) != 0 {
		t.Errorf("expected empty stacks/cgroups")
	}
}

func TestParsePackedSampleWithStacks(t *testing.T) {
	le := binary.LittleEndian
	data := buildMinimalPackedSample()

	// Kern stack: 3 IPs at data offset 0 (24 bytes)
	putSectionDesc(data, sdKernStack, 0, 24)
	// User stack: 2 IPs at data offset 24 (16 bytes)
	putSectionDesc(data, sdUserStack, 24, 16)
	// Cgroups: 2 entries at data offset 40 (16 bytes)
	putSectionDesc(data, sdCgroups, 40, 16)

	// Append data: 3 kern IPs
	for _, ip := range []uint64{0xffffffff80001000, 0xffffffff80002000, 0xffffffff80003000} {
		b := make([]byte, 8)
		le.PutUint64(b, ip)
		data = append(data, b...)
	}
	// 2 user IPs
	for _, ip := range []uint64{0x7f0001000, 0x7f0002000} {
		b := make([]byte, 8)
		le.PutUint64(b, ip)
		data = append(data, b...)
	}
	// 2 cgroups
	for _, cg := range []uint64{100, 200} {
		b := make([]byte, 8)
		le.PutUint64(b, cg)
		data = append(data, b...)
	}

	out := NewRecordSampleParsed()
	if err := ParsePackedSample(data, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out.KernStack) != 3 || out.KernStack[0] != 0xffffffff80001000 {
		t.Errorf("KernStack = %v", out.KernStack)
	}
	if len(out.UserStack) != 2 {
		t.Errorf("UserStack len = %d, want 2", len(out.UserStack))
	}
	if len(out.Cgroups) != 2 || out.Cgroups[0] != 100 || out.Cgroups[1] != 200 {
		t.Errorf("Cgroups = %v", out.Cgroups)
	}
}

func TestParsePackedSampleWithJVM(t *testing.T) {
	le := binary.LittleEndian
	data := buildMinimalPackedSample()

	// User stack: 2 IPs at offset 0
	putSectionDesc(data, sdUserStack, 0, 16)
	// Language sections at offset 16
	langDataSize := langSectionHeaderSize + jvmLangEntrySize // 8 + 16 = 24
	putSectionDesc(data, sdLangSect, 16, uint16(langDataSize))

	// User IPs
	for _, ip := range []uint64{0x7f0001000, 0x7f0002000} {
		b := make([]byte, 8)
		le.PutUint64(b, ip)
		data = append(data, b...)
	}

	// Language section header
	lsh := make([]byte, langSectionHeaderSize)
	le.PutUint16(lsh, uint16(jvmLangEntrySize)) // byte_size
	lsh[2] = 2                                  // language = JVM
	data = append(data, lsh...)

	// JVM entry
	entry := make([]byte, jvmLangEntrySize)
	le.PutUint16(entry[0:2], 1)           // user_stack_index = 1
	le.PutUint64(entry[8:16], 0xdeadbeef) // method_addr
	data = append(data, entry...)

	out := NewRecordSampleParsed()
	if err := ParsePackedSample(data, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.JvmStack.FramesLen != 1 {
		t.Fatalf("JvmStack.FramesLen = %d, want 1", out.JvmStack.FramesLen)
	}
	if out.JvmStack.Frames[0].Index != 1 {
		t.Errorf("Index = %d, want 1", out.JvmStack.Frames[0].Index)
	}
	if out.JvmStack.Frames[0].MethodAddr != 0xdeadbeef {
		t.Errorf("MethodAddr = 0x%x, want 0xdeadbeef", out.JvmStack.Frames[0].MethodAddr)
	}
}

// appendInterpreterFrame appends a single interpreter_frame to dst, encoded in
// little-endian to match the BPF wire format. Layout:
//
//	u64 object_addr | u32 pid | i32 linestart | u64 position_info.
func appendInterpreterFrame(dst []byte, objectAddr uint64, pid uint32, linestart int32, positionInfo uint64) []byte {
	le := binary.LittleEndian
	frame := make([]byte, interpreterFrameSize)
	le.PutUint64(frame[0:8], objectAddr)
	le.PutUint32(frame[8:12], pid)
	le.PutUint32(frame[12:16], uint32(linestart))
	le.PutUint64(frame[16:24], positionInfo)
	return append(dst, frame...)
}

func TestParsePackedSampleWithPythonStack(t *testing.T) {
	le := binary.LittleEndian
	data := buildMinimalPackedSample()

	const numFrames = 2
	langDataSize := langSectionHeaderSize + numFrames*interpreterFrameSize
	putSectionDesc(data, sdLangSect, 0, uint16(langDataSize))

	// Language section header for Python.
	lsh := make([]byte, langSectionHeaderSize)
	le.PutUint16(lsh, uint16(numFrames*interpreterFrameSize))
	lsh[2] = byte(LanguagePython)
	data = append(data, lsh...)

	data = appendInterpreterFrame(data, 0xdeadbeef00000001, 42, 100, 0x7fff00000010)
	data = appendInterpreterFrame(data, 0xdeadbeef00000002, 42, 200, 0x7fff00000020)

	out := NewRecordSampleParsed()
	if err := ParsePackedSample(data, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.PythonStack.Len != numFrames {
		t.Fatalf("PythonStack.Len = %d, want %d", out.PythonStack.Len, numFrames)
	}
	f0 := out.PythonStack.Frames[0]
	if f0.SymbolKey.ObjectAddr != 0xdeadbeef00000001 || f0.SymbolKey.Pid != 42 ||
		f0.SymbolKey.Linestart != 100 || f0.PositionInfo != 0x7fff00000010 {
		t.Errorf("frame0 = %+v", f0)
	}
	f1 := out.PythonStack.Frames[1]
	if f1.SymbolKey.ObjectAddr != 0xdeadbeef00000002 || f1.SymbolKey.Linestart != 200 ||
		f1.PositionInfo != 0x7fff00000020 {
		t.Errorf("frame1 = %+v", f1)
	}
}

func TestParsePackedSampleWithPhpStack(t *testing.T) {
	le := binary.LittleEndian
	data := buildMinimalPackedSample()

	langDataSize := langSectionHeaderSize + interpreterFrameSize
	putSectionDesc(data, sdLangSect, 0, uint16(langDataSize))

	lsh := make([]byte, langSectionHeaderSize)
	le.PutUint16(lsh, uint16(interpreterFrameSize))
	lsh[2] = byte(LanguagePhp)
	data = append(data, lsh...)

	// PHP leaves position_info at 0.
	data = appendInterpreterFrame(data, 0xc0ffee00, 7, 55, 0)

	out := NewRecordSampleParsed()
	if err := ParsePackedSample(data, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.PhpStack.Len != 1 {
		t.Fatalf("PhpStack.Len = %d, want 1", out.PhpStack.Len)
	}
	got := out.PhpStack.Frames[0]
	if got.SymbolKey.ObjectAddr != 0xc0ffee00 || got.SymbolKey.Pid != 7 ||
		got.SymbolKey.Linestart != 55 || got.PositionInfo != 0 {
		t.Errorf("frame = %+v", got)
	}
}

func TestParsePackedSampleTooShort(t *testing.T) {
	out := NewRecordSampleParsed()
	if err := ParsePackedSample(make([]byte, 10), out); err == nil {
		t.Fatal("expected error for short data")
	}
}

func TestParsePackedSample_Reuse_ClearSlices(t *testing.T) {
	le := binary.LittleEndian
	out := NewRecordSampleParsed()

	// 1. Parse a sample with stacks
	dataWithStacks := buildMinimalPackedSample()
	putSectionDesc(dataWithStacks, sdKernStack, 0, 24)
	putSectionDesc(dataWithStacks, sdUserStack, 24, 16)
	for _, ip := range []uint64{1, 2, 3, 4, 5} {
		b := make([]byte, 8)
		le.PutUint64(b, ip)
		dataWithStacks = append(dataWithStacks, b...)
	}

	if err := ParsePackedSample(dataWithStacks, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.KernStack) == 0 || len(out.UserStack) == 0 {
		t.Fatalf("expected stacks to be populated")
	}

	// 2. Parse a minimal sample (no stacks) into the SAME object
	dataMinimal := buildMinimalPackedSample()
	if err := ParsePackedSample(dataMinimal, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3. Verify slices are cleared
	if len(out.KernStack) != 0 {
		t.Errorf("KernStack not cleared, len = %d", len(out.KernStack))
	}
	if len(out.UserStack) != 0 {
		t.Errorf("UserStack not cleared, len = %d", len(out.UserStack))
	}
	if len(out.Cgroups) != 0 {
		t.Errorf("Cgroups not cleared, len = %d", len(out.Cgroups))
	}
}

func TestParsePackedSample_Reuse_Resize(t *testing.T) {
	le := binary.LittleEndian
	out := NewRecordSampleParsed()

	// 1. Parse a sample with a long stack (3 frames)
	dataLong := buildMinimalPackedSample()
	putSectionDesc(dataLong, sdKernStack, 0, 24)
	for _, ip := range []uint64{1, 2, 3} {
		b := make([]byte, 8)
		le.PutUint64(b, ip)
		dataLong = append(dataLong, b...)
	}
	if err := ParsePackedSample(dataLong, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2. Parse a sample with a short stack (2 frames) into the SAME object
	dataShort := buildMinimalPackedSample()
	putSectionDesc(dataShort, sdKernStack, 0, 16)
	for _, ip := range []uint64{99, 100} {
		b := make([]byte, 8)
		le.PutUint64(b, ip)
		dataShort = append(dataShort, b...)
	}
	if err := ParsePackedSample(dataShort, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3. Verify length and content
	if len(out.KernStack) != 2 {
		t.Fatalf("KernStack len = %d, want 2", len(out.KernStack))
	}
	if out.KernStack[0] != 99 || out.KernStack[1] != 100 {
		t.Errorf("KernStack data incorrect: %v", out.KernStack)
	}
}

func BenchmarkParsePackedSampleMinimal(b *testing.B) {
	data := buildMinimalPackedSample()
	out := NewRecordSampleParsed()
	b.ResetTimer()
	for b.Loop() {
		_ = ParsePackedSample(data, out)
	}
}
