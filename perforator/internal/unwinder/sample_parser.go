package unwinder

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"
)

var le = binary.LittleEndian

// Sizes derived from generated types — no hand-maintained constants needed.
// The btf2go-generated layout-check var block in unwinder.go enforces that
// these match the BPF C struct sizes at Go compile time.
// Sizes derived from generated types — no hand-maintained constants needed.
// The btf2go-generated layout-check var block in unwinder.go enforces that
// these match the BPF C struct sizes at Go compile time.
var (
	recordSampleHeaderSize = int(unsafe.Sizeof(RecordSampleHeader{}))
	tlsResultSize          = int(unsafe.Sizeof(ThreadLocalVariableCollectResult{}))
	branchRecordSize       = int(unsafe.Sizeof(BranchRecord{}))
	interpreterFrameSize   = int(unsafe.Sizeof(InterpreterFrame{}))
	jvmLangEntrySize       = int(unsafe.Sizeof(JvmLangEntry{}))
	langSectionHeaderSize  = int(unsafe.Sizeof(LanguageSectionHeader{}))
)

var (
	errTooShort      = errors.New("packed sample data too short")
	errBadSectionLen = errors.New("language section byte_size exceeds remaining data")
)

// sectionSlice returns the byte slice for a section, or nil if empty/out of bounds.
func sectionSlice(data []byte, base int, off, size uint16) ([]byte, error) {
	if size == 0 {
		return nil, nil
	}
	start := base + int(off)
	end := start + int(size)
	if start > len(data) || end > len(data) {
		return nil, fmt.Errorf("%w: offset=%d size=%d", errTooShort, off, size)
	}
	return data[start:end], nil
}

func ParsePackedSample(data []byte, out *RecordSampleParsed) error {
	if len(data) < recordSampleHeaderSize {
		return fmt.Errorf("%w: need %d bytes, got %d", errTooShort, recordSampleHeaderSize, len(data))
	}

	if err := out.RecordSampleHeader.UnmarshalBinaryUnsafe(data); err != nil {
		return err
	}

	// Clear slices to avoid leaking data from previous samples when reusing `out`.
	out.KernStack = out.KernStack[:0]
	out.UserStack = out.UserStack[:0]
	out.Cgroups = out.Cgroups[:0]
	out.LBR = out.LBR[:0]
	out.TLS = out.TLS[:0]

	base := recordSampleHeaderSize

	if off, sz := out.GetSection(RecordSampleHeaderSectionKernStack); sz > 0 {
		if err := parseU64s(data, base, off, sz, &out.KernStack); err != nil {
			return err
		}
	}
	if off, sz := out.GetSection(RecordSampleHeaderSectionUserStack); sz > 0 {
		if err := parseU64s(data, base, off, sz, &out.UserStack); err != nil {
			return err
		}
	}
	if off, sz := out.GetSection(RecordSampleHeaderSectionCgroups); sz > 0 {
		if err := parseU64s(data, base, off, sz, &out.Cgroups); err != nil {
			return err
		}
	}
	if off, sz := out.GetSection(RecordSampleHeaderSectionLbr); sz > 0 {
		if err := parseLBR(data, base, off, sz, &out.LBR); err != nil {
			return err
		}
	}
	if off, sz := out.GetSection(RecordSampleHeaderSectionTls); sz > 0 {
		if err := parseTLS(data, base, off, sz, &out.TLS); err != nil {
			return err
		}
	}

	out.PythonStack.Len = 0
	out.PhpStack.Len = 0
	out.JvmStack.FramesLen = 0
	off, sz := out.GetSection(RecordSampleHeaderSectionLanguageSections)
	return parseLangSections(data, base, off, sz, out)
}

func parseU64s(data []byte, base int, off, size uint16, out *[]uint64) error {
	raw, err := sectionSlice(data, base, off, size)
	if err != nil {
		return err
	}
	n := len(raw) / 8
	if cap(*out) >= n {
		*out = (*out)[:n]
	} else {
		*out = make([]uint64, n)
	}
	for i := 0; i < n; i++ {
		(*out)[i] = le.Uint64(raw[i*8 : i*8+8])
	}
	return nil
}

func parseLBR(data []byte, base int, off, size uint16, out *[]BranchRecord) error {
	raw, err := sectionSlice(data, base, off, size)
	if err != nil {
		return err
	}
	n := len(raw) / branchRecordSize
	if cap(*out) >= n {
		*out = (*out)[:n]
	} else {
		*out = make([]BranchRecord, n)
	}
	for i := 0; i < n; i++ {
		o := i * branchRecordSize
		(*out)[i].From = le.Uint64(raw[o : o+8])
		(*out)[i].To = le.Uint64(raw[o+8 : o+16])
		(*out)[i].Flags = le.Uint64(raw[o+16 : o+24])
	}
	return nil
}

func parseTLS(data []byte, base int, off, size uint16, out *[]ThreadLocalVariableCollectResult) error {
	raw, err := sectionSlice(data, base, off, size)
	if err != nil {
		return err
	}
	n := len(raw) / tlsResultSize
	if cap(*out) >= n {
		*out = (*out)[:n]
	} else {
		*out = make([]ThreadLocalVariableCollectResult, n)
	}
	for i := 0; i < n; i++ {
		o := i * tlsResultSize
		(*out)[i].Offset = le.Uint64(raw[o : o+8])
		(*out)[i].Type = TlsVariableType(raw[o+8])
		copy((*out)[i].Value.UnionBuf[:], raw[o+16:o+tlsResultSize])
	}
	return nil
}

func parseLangSections(data []byte, base int, off, size uint16, out *RecordSampleParsed) error {
	raw, err := sectionSlice(data, base, off, size)
	if err != nil || raw == nil {
		return err
	}
	pos := 0
	for pos+langSectionHeaderSize <= len(raw) {
		byteSize := int(le.Uint16(raw[pos:]))
		language := LanguageId(raw[pos+2])
		pos += langSectionHeaderSize
		if pos+byteSize > len(raw) {
			return fmt.Errorf("%w: byte_size=%d", errBadSectionLen, byteSize)
		}
		frames := raw[pos : pos+byteSize]
		switch language {
		case LanguagePython:
			decodeInterpreterFrames(frames, &out.PythonStack)
		case LanguagePhp:
			decodeInterpreterFrames(frames, &out.PhpStack)
		case LanguageJvm:
			decodeJvmEntries(frames, &out.JvmStack)
		case LanguageLua:
			decodeInterpreterFrames(frames, &out.LuaStack)
		}
		pos += byteSize
	}
	return nil
}

func decodeInterpreterFrames(data []byte, stack *InterpreterStack) {
	n := len(data) / interpreterFrameSize
	if n > len(stack.Frames) {
		n = len(stack.Frames)
	}
	stack.Len = uint8(n)
	for i := 0; i < n; i++ {
		off := i * interpreterFrameSize
		stack.Frames[i].SymbolKey.ObjectAddr = le.Uint64(data[off : off+8])
		stack.Frames[i].SymbolKey.Pid = le.Uint32(data[off+8 : off+12])
		stack.Frames[i].SymbolKey.Linestart = int32(le.Uint32(data[off+12 : off+16]))
	}
}

func decodeJvmEntries(data []byte, jstack *JvmStack) {
	n := len(data) / jvmLangEntrySize
	if n > len(jstack.Frames) {
		n = len(jstack.Frames)
	}
	jstack.FramesLen = uint32(n)
	for i := 0; i < n; i++ {
		off := i * jvmLangEntrySize
		jstack.Frames[i].Index = uint32(le.Uint16(data[off : off+2]))
		jstack.Frames[i].MethodAddr = le.Uint64(data[off+8 : off+16])
	}
}
