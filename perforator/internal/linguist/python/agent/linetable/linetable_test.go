package linetable

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// encodeFirstByte builds an entry's leading byte.
// code in [0, 15], lengthCodeUnits in [1, 8].
func encodeFirstByte(code uint8, lengthCodeUnits uint8) byte {
	return 0x80 | ((code & 0xF) << 3) | ((lengthCodeUnits - 1) & 0x7)
}

// encodeUnsignedVarint encodes a uint32 as the LSB-first 6-bit chunked
// varint with continuation bit on non-final bytes.
func encodeUnsignedVarint(v uint32) []byte {
	var out []byte
	for {
		chunk := byte(v & 0x3F)
		v >>= 6
		if v != 0 {
			chunk |= 0x40 // continuation bit
		}
		out = append(out, chunk)
		if v == 0 {
			break
		}
	}
	return out
}

// encodeSignedVarint encodes an int32 by CPython's convention: low bit = sign
// (1 means negative), upper bits = abs value, then through encodeUnsignedVarint.
func encodeSignedVarint(v int32) []byte {
	var u uint32
	if v < 0 {
		u = (uint32(-v) << 1) | 1
	} else {
		u = uint32(v) << 1
	}
	return encodeUnsignedVarint(u)
}

// shortEntry builds a SHORT (code 0-9) entry: first byte + 1 column byte.
func shortEntry(code uint8, lengthCodeUnits uint8, col byte) []byte {
	return []byte{encodeFirstByte(code, lengthCodeUnits), col}
}

func TestInstrPtrToBytecodeOffset_OK(t *testing.T) {
	cases := []struct {
		name           string
		instrPtr       uint64
		codeObjectAddr uint64
		coCodeAdaptive uint64
		wantOffset     int32
		wantOK         bool
	}{
		{
			name:           "normal_case",
			instrPtr:       0x10A4,
			codeObjectAddr: 0x1000,
			coCodeAdaptive: 0x80,
			wantOffset:     0x24, // 36
			wantOK:         true,
		},
		{
			name:           "instrPtr_below_base",
			instrPtr:       0x1000,
			codeObjectAddr: 0x1000,
			coCodeAdaptive: 0x80,
			wantOffset:     0,
			wantOK:         false,
		},
		{
			name:           "diff_exceeds_limit",
			instrPtr:       0x1000 + 0x80 + uint64(1<<24) + 1,
			codeObjectAddr: 0x1000,
			coCodeAdaptive: 0x80,
			wantOffset:     0,
			wantOK:         false,
		},
		{
			name:           "instrPtr_equal_to_base",
			instrPtr:       0x1000 + 0x80,
			codeObjectAddr: 0x1000,
			coCodeAdaptive: 0x80,
			wantOffset:     0,
			wantOK:         true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			off, ok := InstrPtrToBytecodeOffset(tc.instrPtr, tc.codeObjectAddr, tc.coCodeAdaptive)
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.wantOffset, off)
		})
	}
}

func TestUnmarshal_Empty(t *testing.T) {
	_, err := Unmarshal(nil, 5)
	require.ErrorIs(t, err, ErrEmptyLinetable)

	_, err = Unmarshal([]byte{}, 5)
	require.ErrorIs(t, err, ErrEmptyLinetable)
}

func TestResolveLine_SingleShortEntry(t *testing.T) {
	// SHORT entry, code=0, length=3 code units (6 bytes), col=0.
	// Entry bytes: [encodeFirstByte(0,3), 0x00]
	raw := shortEntry(0, 3, 0x00)
	table := LocationTable{FirstLineno: 10, Raw: raw}

	// Offsets within [0, 6) → (10, true).
	for _, off := range []int32{0, 2, 4} {
		line, ok := table.ResolveLine(off)
		require.True(t, ok, "offset %d should match", off)
		require.Equal(t, int32(10), line, "offset %d", off)
	}

	// Offset 6 → past end → (0, false).
	line, ok := table.ResolveLine(6)
	require.False(t, ok)
	require.Equal(t, int32(0), line)
}

func TestResolveLine_OneLineForms(t *testing.T) {
	// Entry 1: SHORT (code=0, length=1). Byte range [0, 2). Delta=0.
	// Entry 2: ONE_LINE_1 (code=11, length=2). Byte range [2, 6). Delta=1.
	//          First byte + 2 extra bytes (col_start, col_end).
	// Entry 3: ONE_LINE_2 (code=12, length=1). Byte range [6, 8). Delta=2.
	//          First byte + 2 extra bytes.
	raw := []byte{
		encodeFirstByte(0, 1), 0x00, // SHORT, 1 code unit, col=0
		encodeFirstByte(11, 2), 0x01, 0x02, // ONE_LINE_1 (delta=1), 2 code units
		encodeFirstByte(12, 1), 0x03, 0x04, // ONE_LINE_2 (delta=2), 1 code unit
	}
	table := LocationTable{FirstLineno: 100, Raw: raw}

	cases := []struct {
		offset   int32
		wantLine int32
		wantOK   bool
	}{
		{0, 100, true},
		{1, 100, true},
		{2, 101, true},
		{5, 101, true},
		{6, 103, true},
		{8, 0, false},
	}

	for _, tc := range cases {
		line, ok := table.ResolveLine(tc.offset)
		require.Equal(t, tc.wantOK, ok, "offset %d", tc.offset)
		if tc.wantOK {
			require.Equal(t, tc.wantLine, line, "offset %d", tc.offset)
		}
	}
}

func TestResolveLine_NoColumns_SignedVarint(t *testing.T) {
	// Entry code=13 (NO_COLUMNS), length=1 code unit, line delta=-3.
	// Byte range [0, 2).
	delta := encodeSignedVarint(-3)
	raw := append([]byte{encodeFirstByte(13, 1)}, delta...)
	table := LocationTable{FirstLineno: 50, Raw: raw}

	line, ok := table.ResolveLine(0)
	require.True(t, ok)
	require.Equal(t, int32(47), line)

	line, ok = table.ResolveLine(2)
	require.False(t, ok)
	require.Equal(t, int32(0), line)
}

func TestResolveLine_LongEntry_PositiveDelta(t *testing.T) {
	// Entry code=14 (LONG), length=2 code units. Byte range [0, 4).
	// Line delta=7 (signed varint), then 3 unsigned varints: 1, 2, 3.
	raw := []byte{encodeFirstByte(14, 2)}
	raw = append(raw, encodeSignedVarint(7)...)
	raw = append(raw, encodeUnsignedVarint(1)...)
	raw = append(raw, encodeUnsignedVarint(2)...)
	raw = append(raw, encodeUnsignedVarint(3)...)
	table := LocationTable{FirstLineno: 200, Raw: raw}

	line, ok := table.ResolveLine(0)
	require.True(t, ok)
	require.Equal(t, int32(207), line)

	line, ok = table.ResolveLine(4)
	require.False(t, ok)
	require.Equal(t, int32(0), line)
}

func TestResolveLine_NoneEntry_ReturnsFalse(t *testing.T) {
	// Entry code=15 (NONE), length=4 code units. Byte range [0, 8).
	raw := []byte{encodeFirstByte(15, 4)}
	table := LocationTable{FirstLineno: 42, Raw: raw}

	line, ok := table.ResolveLine(0)
	require.False(t, ok)
	require.Equal(t, int32(0), line)
}

func TestResolveLine_OffsetPastEnd(t *testing.T) {
	// Single SHORT entry, length=1 code unit. Byte range [0, 2).
	raw := shortEntry(0, 1, 0x00)
	table := LocationTable{FirstLineno: 10, Raw: raw}

	line, ok := table.ResolveLine(1 << 20)
	require.False(t, ok)
	require.Equal(t, int32(0), line)
}

func TestResolveLine_OffsetNegative(t *testing.T) {
	// Single SHORT entry, length=1 code unit.
	raw := shortEntry(0, 1, 0x00)
	table := LocationTable{FirstLineno: 10, Raw: raw}

	line, ok := table.ResolveLine(-1)
	require.False(t, ok)
	require.Equal(t, int32(0), line)
}

func TestResolveLine_NegativeFirstLineno(t *testing.T) {
	// Entry code=13 (NO_COLUMNS), length=1 code unit, line delta=-5.
	// FirstLineno=2. Expected: 2 + (-5) = -3.
	delta := encodeSignedVarint(-5)
	raw := append([]byte{encodeFirstByte(13, 1)}, delta...)
	table := LocationTable{FirstLineno: 2, Raw: raw}

	line, ok := table.ResolveLine(0)
	require.True(t, ok)
	require.Equal(t, int32(-3), line)
}
