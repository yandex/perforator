// Package linetable decodes CPython 3.11+ co_linetable bytes and resolves
// bytecode offsets to source line numbers. See Objects/locations.md and
// Objects/codeobject.c:advance_with_locations in the CPython source tree.
package linetable

import (
	"errors"
)

// CPython 3.11+ code units are two bytes each (one wordcode instruction).
const codeUnitBytes = 2

// _PyCodeLocationInfoKind values — see Include/internal/pycore_code.h.
const (
	locInfoOneLine0   = 10
	locInfoOneLine2   = 12
	locInfoNoColumns  = 13
	locInfoLong       = 14
	locInfoNoLocation = 15
)

// ErrEmptyLinetable is returned by Unmarshal when given a zero-length slice.
var ErrEmptyLinetable = errors.New("python: empty co_linetable")

// LocationTable holds a parsed co_linetable. Raw is walked lazily.
// Unresolvable marks negative cache entries; see [Cache.AddTombstone].
type LocationTable struct {
	FirstLineno  int32
	Raw          []byte
	Unresolvable bool
}

// Unmarshal wraps the raw co_linetable bytes in a LocationTable.
func Unmarshal(data []byte, firstLineno int32) (LocationTable, error) {
	if len(data) == 0 {
		return LocationTable{}, ErrEmptyLinetable
	}
	return LocationTable{FirstLineno: firstLineno, Raw: data}, nil
}

// bytecodeRangeLimit guards InstrPtrToBytecodeOffset against absurd deltas
// that signal a stale BPF-captured pointer.
const bytecodeRangeLimit = int32(1 << 24)

// InstrPtrToBytecodeOffset converts a BPF-captured instruction pointer into
// a byte offset in the bytecode array. Returns (0, false) on negative or
// implausibly large differences.
//
// On CPython 3.11/3.12 the BPF-captured value is `_PyInterpreterFrame.prev_instr`,
// which is initialized to `co_code_adaptive - 1` (one code unit before the first
// instruction) until the first opcode runs. That underflow of exactly
// [codeUnitBytes] is mapped to bytecode offset 0 so short-lived callees still
// resolve to a line. Larger underflows are rejected.
//
// Free-threaded CPython (3.13t+) is unsupported: instr_ptr may point into
// TLBC rather than co_code_adaptive.
func InstrPtrToBytecodeOffset(instrPtr, codeObjectAddr, coCodeAdaptive uint64) (int32, bool) {
	base := codeObjectAddr + coCodeAdaptive
	if instrPtr < base {
		if base-instrPtr == uint64(codeUnitBytes) {
			return 0, true
		}
		return 0, false
	}
	diff := instrPtr - base
	if diff > uint64(bytecodeRangeLimit) {
		return 0, false
	}
	return int32(diff), true
}

// scanSignedVarint decodes one signed varint. Encoding matches CPython's
// scan_varint / scan_signed_varint: 6-bit chunks, LSB first; low bit of the
// unsigned value is the sign (odd → negative), not zigzag.
func scanSignedVarint(buf []byte, off int) (val int32, n int, ok bool) {
	u, n, ok := scanVarint(buf, off)
	if !ok {
		return 0, 0, false
	}
	if u&1 != 0 {
		return -int32(u >> 1), n, true
	}
	return int32(u >> 1), n, true
}

func scanVarint(buf []byte, off int) (val uint32, n int, ok bool) {
	var shift uint
	for {
		if off+n >= len(buf) {
			return 0, 0, false
		}
		b := buf[off+n]
		n++
		val |= uint32(b&63) << shift
		if b&64 == 0 {
			return val, n, true
		}
		shift += 6
		// 5 chunks × 6 bits cover uint32; longer is malformed.
		if shift >= 32 {
			return 0, 0, false
		}
	}
}

// lineDelta returns the start-line delta for the entry at entryStart
// (CPython get_line_delta), with bounds checks for truncated tables.
func lineDelta(buf []byte, entryStart int) (delta int32, ok bool) {
	code := (buf[entryStart] >> 3) & 0xF
	switch {
	case code == locInfoNoLocation:
		return 0, true
	case code == locInfoNoColumns, code == locInfoLong:
		d, _, vok := scanSignedVarint(buf, entryStart+1)
		if !vok {
			return 0, false
		}
		return d, true
	case code >= locInfoOneLine0 && code <= locInfoOneLine2:
		return int32(code) - 10, true
	default:
		// Short forms (0..9): no line delta.
		return 0, true
	}
}

// ResolveLine returns the source line for a bytecode byte offset.
// ok=false when the offset is past the covered range or the entry has no
// location (PY_CODE_LOCATION_INFO_NONE).
func (table LocationTable) ResolveLine(bytecodeOffset int32) (int32, bool) {
	if bytecodeOffset < 0 || len(table.Raw) == 0 {
		return 0, false
	}

	buf := table.Raw
	pos := 0
	currentLine := table.FirstLineno
	var byteEnd int32

	for pos < len(buf) {
		first := buf[pos]
		if first&0x80 == 0 {
			// Not on an entry start — table is malformed or cursor desynced.
			return 0, false
		}
		code := (first >> 3) & 0xF
		length := int32((first & 7) + 1) // in code units

		delta, ok := lineDelta(buf, pos)
		if !ok {
			return 0, false
		}
		currentLine += delta

		byteStart := byteEnd
		byteEnd = byteStart + length*codeUnitBytes

		if bytecodeOffset >= byteStart && bytecodeOffset < byteEnd {
			if code == locInfoNoLocation {
				return 0, false
			}
			return currentLine, true
		}

		// Skip payload until the next entry start (bit 7 set).
		pos++
		for pos < len(buf) && buf[pos]&0x80 == 0 {
			pos++
		}
	}

	return 0, false
}
