package proto

import (
	"bytes"
	"io"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ClickHouse/ch-go/internal/gold"
)

func TestColBFloat16_DecodeColumn(t *testing.T) {
	t.Parallel()
	src := rand.NewSource(99) // don't use random number like timestamp. That would mess up "golden" file everytime
	rng := rand.New(src)

	// BFloat16 may have precision loss, so check with tolerance
	// BFloat16 has 7-bit for mantissa compared to 23-bit mantissa for Float32.
	// which makes it loose 3-4 digits of precision.
	relativeError := 0.004 // 0.4%

	const rows = 5000
	var data ColBFloat16
	for i := range rows {
		v := rng.Float32() * 1000 // range [0..1000]
		data.Append(v)
		maxDelta := math.Abs(float64(v)) * relativeError
		require.InDelta(t, v, data.Row(i), maxDelta)
	}

	var buf Buffer
	data.EncodeColumn(&buf)
	t.Run("Golden", func(t *testing.T) {
		t.Parallel()
		gold.Bytes(t, buf.Buf, "col_bfloat16")
	})
	t.Run("HappyPath", func(t *testing.T) {
		br := bytes.NewReader(buf.Buf)
		r := NewReader(br)

		var dec ColBFloat16
		require.NoError(t, dec.DecodeColumn(r, rows))
		require.Equal(t, data, dec)
		require.Equal(t, rows, dec.Rows())
		dec.Reset()
		require.Equal(t, 0, dec.Rows())

		require.Equal(t, ColumnTypeBFloat16, dec.Type())
	})
	t.Run("ZeroRows", func(t *testing.T) {
		r := NewReader(bytes.NewReader(nil))

		var dec ColBFloat16
		require.NoError(t, dec.DecodeColumn(r, 0))
	})
	t.Run("EOF", func(t *testing.T) {
		r := NewReader(bytes.NewReader(nil))

		var dec ColBFloat16
		require.ErrorIs(t, dec.DecodeColumn(r, rows), io.EOF)
	})
	t.Run("NoShortRead", func(t *testing.T) {
		var dec ColBFloat16
		requireNoShortRead(t, buf.Buf, colAware(&dec, rows))
	})
	t.Run("ZeroRowsEncode", func(t *testing.T) {
		var v ColBFloat16
		v.EncodeColumn(nil) // should be no-op
	})
	t.Run("WriteColumn", checkWriteColumn(data))
}

func TestBFloat16_Row(t *testing.T) {
	t.Parallel()
	var col ColBFloat16

	testCases := []struct {
		name     string
		value    float32
		expected float32
		delta    float64
	}{
		{"zero", 0.0, 0.0, 0.0},
		{"one", 1.0, 1.0, 0.0},
		{"negative_one", -1.0, -1.0, 0.0},
		{"small_positive", 0.125, 0.125, 0.001},
		{"small_negative", -0.125, -0.125, 0.001},
		{"pi", 3.14159, 3.14159, 0.01},
		{"large", 12345.0, 12345.0, 10.0}, // BFloat16 has limited precision for large values
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			col.Append(tc.value)
			result := col.Row(len(col) - 1)
			require.InDelta(t, tc.expected, result, tc.delta, "BFloat16 conversion precision")
		})
	}
}

func TestBFloat16_Append(t *testing.T) {
	t.Parallel()
	var col ColBFloat16

	values := []float32{0.0, 1.0, -1.0, 3.14, 2.718, 100.5, -50.25}
	col.AppendArr(values)

	require.Equal(t, len(values), col.Rows())
	for i, expected := range values {
		result := col.Row(i)
		require.InDelta(t, expected, result, 0.01, "BFloat16 precision at index %d", i)
	}
}

func TestBFloat16_EdgeCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   float32
		checkFn func(t *testing.T, result float32)
	}{
		{
			"positive_zero",
			0.0,
			func(t *testing.T, result float32) {
				require.Equal(t, float32(0.0), result)
			},
		},
		{
			"negative_zero",
			float32(math.Copysign(0, -1)),
			func(t *testing.T, result float32) {
				// Negative zero should be preserved
				require.True(t, math.Signbit(float64(result)), "should be negative zero")
			},
		},
		{
			"positive_infinity",
			float32(math.Inf(1)),
			func(t *testing.T, result float32) {
				require.True(t, math.IsInf(float64(result), 1), "should be positive infinity")
			},
		},
		{
			"negative_infinity",
			float32(math.Inf(-1)),
			func(t *testing.T, result float32) {
				require.True(t, math.IsInf(float64(result), -1), "should be negative infinity")
			},
		},
		{
			"nan",
			float32(math.NaN()),
			func(t *testing.T, result float32) {
				require.True(t, math.IsNaN(float64(result)), "should be NaN")
			},
		},
		{
			"very_small_positive",
			1e-6,
			func(t *testing.T, result float32) {
				require.InDelta(t, 1e-6, result, 1e-7)
			},
		},
		{
			"very_small_negative",
			-1e-6,
			func(t *testing.T, result float32) {
				require.InDelta(t, -1e-6, result, 1e-7)
			},
		},
		{
			"very_large_positive",
			1.0e30,
			func(t *testing.T, result float32) {
				require.InDelta(t, 1.0e30, result, 1e27)
			},
		},
		{
			"very_large_negative",
			-1.0e30,
			func(t *testing.T, result float32) {
				require.InDelta(t, -1.0e30, result, 1e27)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var col ColBFloat16
			col.Append(tc.value)
			result := col.Row(0)
			tc.checkFn(t, result)
		})
	}
}

func TestBFloat16_Precision(t *testing.T) {
	t.Parallel()

	// BFloat16 has ~3.5 significant decimal digits of precision
	// Test precision boundaries
	testCases := []struct {
		name     string
		input    float32
		expected float32
		maxDelta float64
	}{
		// Values that should roundtrip exactly (powers of 2)
		{"power_of_2_1", 1.0, 1.0, 0.0},
		{"power_of_2_2", 2.0, 2.0, 0.0},
		{"power_of_2_4", 4.0, 4.0, 0.0},
		{"power_of_2_8", 8.0, 8.0, 0.0},
		{"power_of_2_half", 0.5, 0.5, 0.0},
		{"power_of_2_quarter", 0.25, 0.25, 0.0},

		// Values with precision loss
		{"pi", 3.14159265, 3.140625, 0.002}, // BFloat16 precision
		{"e", 2.71828183, 2.71875, 0.001},
		{"fraction", 0.1, 0.1, 0.001},
		{"decimal", 1.23456789, 1.234375, 0.001},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var col ColBFloat16
			col.Append(tc.input)
			result := col.Row(0)
			require.InDelta(t, tc.expected, result, tc.maxDelta,
				"input=%v, expected=%v, got=%v", tc.input, tc.expected, result)
		})
	}
}

func TestBFloat16_RoundTrip(t *testing.T) {
	t.Parallel()

	// Test encode -> decode -> verify
	const rows = 100
	var original ColBFloat16

	// Create test data with various values
	for i := range rows {
		v := float32(i)*0.5 - 25.0 // Range: -25.0 to 24.5
		original.Append(v)
	}

	// Encode
	var buf Buffer
	original.EncodeColumn(&buf)

	// Decode
	r := NewReader(bytes.NewReader(buf.Buf))
	var decoded ColBFloat16
	require.NoError(t, decoded.DecodeColumn(r, rows))

	// Verify
	require.Equal(t, original.Rows(), decoded.Rows())
	for i := range rows {
		require.Equal(t, original[i], decoded[i],
			"mismatch at index %d: original=%v, decoded=%v",
			i, original[i], decoded[i])
	}
}

func TestBFloat16_Conversion(t *testing.T) {
	t.Parallel()

	// Test Float32 -> BFloat16 -> Float32 conversion accuracy
	testCases := []float32{
		0.0,
		1.0,
		-1.0,
		0.5,
		-0.5,
		100.0,
		-100.0,
		0.125,
		3.14159,
		2.71828,
		float32(math.Inf(1)),
		float32(math.Inf(-1)),
		float32(math.NaN()),
	}

	for _, input := range testCases {
		t.Run("", func(t *testing.T) {
			var col ColBFloat16
			col.Append(input)
			output := col.Row(0)

			if math.IsNaN(float64(input)) {
				require.True(t, math.IsNaN(float64(output)), "NaN should remain NaN")
			} else if math.IsInf(float64(input), 0) {
				require.True(t, math.IsInf(float64(output), int(math.Copysign(1, float64(input)))),
					"Inf should remain Inf with same sign")
			} else {
				// For normal values, check relative error
				// BFloat16 has ~7 bits of mantissa precision
				relativeError := math.Abs(float64(output-input)) / math.Max(math.Abs(float64(input)), 1e-6)
				require.Less(t, relativeError, 0.01, "relative error too large for input %v: got %v", input, output)
			}
		})
	}
}

func BenchmarkBFloat16_Decode(b *testing.B) {
	const rows = 1000
	var data ColBFloat16
	for i := range rows {
		data.Append(float32(i) * 0.1)
	}

	var buf Buffer
	data.EncodeColumn(&buf)

	b.ReportAllocs()
	b.SetBytes(int64(len(buf.Buf)))

	for b.Loop() {
		br := bytes.NewReader(buf.Buf)
		r := NewReader(br)

		var dec ColBFloat16
		if err := dec.DecodeColumn(r, rows); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBFloat16_Encode(b *testing.B) {
	const rows = 1000
	var data ColBFloat16
	for i := range rows {
		data.Append(float32(i) * 0.1)
	}

	b.ReportAllocs()

	for b.Loop() {
		var buf Buffer
		data.EncodeColumn(&buf)
		b.SetBytes(int64(len(buf.Buf)))
	}
}

func BenchmarkBFloat16_Append(b *testing.B) {
	var col ColBFloat16

	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		col.Append(float32(i) * 0.1)
	}
}

func BenchmarkBFloat16_Row(b *testing.B) {
	const rows = 1000
	var col ColBFloat16
	for i := range rows {
		col.Append(float32(i) * 0.1)
	}

	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		_ = col.Row(i % rows)
	}
}

// TestBFloat16_ExhaustiveBFloat16ToFloat32 tests all 65,536 possible BFloat16 values
// converting to Float32 and back to BFloat16. This ensures perfect round-trip for
// all valid BFloat16 values.
func TestBFloat16_ExhaustiveBFloat16ToFloat32(t *testing.T) {
	// Test all 65,536 possible BFloat16 values (2^16)
	// Use int to avoid overflow issues with uint16
	for i := 0; i <= 65535; i++ {
		bfloat16Val := uint16(i)

		// Create a BFloat16 value directly
		var col ColBFloat16
		col = append(col, bfloat16Val)

		// Convert to Float32
		f32 := col.Row(0)

		// Convert back to BFloat16
		var col2 ColBFloat16
		col2.Append(f32)

		// The round-trip should be exact
		if col2[0] != bfloat16Val {
			t.Errorf("BFloat16 round-trip failed for value 0x%04x: got 0x%04x, float32 intermediate = %v (%08x)",
				bfloat16Val, col2[0], f32, math.Float32bits(f32))
		}

		// Also verify that Row() correctly reconstructs the float32
		// by checking the bit pattern
		expectedBits := uint32(bfloat16Val) << 16
		actualBits := math.Float32bits(f32)
		if actualBits != expectedBits {
			t.Errorf("BFloat16 to Float32 conversion failed for value 0x%04x: expected bits 0x%08x, got 0x%08x",
				bfloat16Val, expectedBits, actualBits)
		}
	}
}

// TestBFloat16_ExhaustiveFloat32ToBFloat16 tests all 4,294,967,296 possible Float32 values
// converting to BFloat16. This verifies that the round-to-nearest-even rounding is
// correctly implemented for all possible inputs.
// This test is skipped in short mode as it takes ~60-90 seconds to complete.
func TestBFloat16_ExhaustiveFloat32ToBFloat16(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping exhaustive Float32 to BFloat16 test in short mode (takes ~60-90s)")
	}

	// Helper function to compute expected BFloat16 value with round-to-nearest-even
	expectedBFloat16 := func(f32Bits uint32) uint16 {
		// Extract components
		upper16 := f32Bits >> 16
		lower16 := f32Bits & 0xFFFF

		// Round-to-nearest-even logic
		// If lower16 > 0x8000: round up
		// If lower16 < 0x8000: round down (truncate)
		// If lower16 == 0x8000: round to even (check LSB of upper16)
		if lower16 > 0x8000 {
			// Round up
			upper16++
		} else if lower16 == 0x8000 {
			// Exactly at midpoint - round to even
			// If upper16 is odd (LSB=1), round up to make it even
			if (upper16 & 1) == 1 {
				upper16++
			}
		}
		// else: lower16 < 0x8000, round down (do nothing, just truncate is enough)

		return uint16(upper16)
	}

	// Test all 4,294,967,296 possible Float32 bit patterns
	var f32Bits uint32
	failCount := 0
	const maxFailuresToReport = 10

	for {
		f32 := math.Float32frombits(f32Bits)

		// Convert to BFloat16
		var col ColBFloat16
		col.Append(f32)
		actual := col[0]

		// Compute expected value
		expected := expectedBFloat16(f32Bits)

		if actual != expected {
			if failCount < maxFailuresToReport {
				t.Errorf("Float32 to BFloat16 conversion failed for f32=0x%08x (%v): expected 0x%04x, got 0x%04x",
					f32Bits, f32, expected, actual)
			}
			failCount++
		}

		// Progress reporting every 100 million iterations (~2.3% intervals)
		if f32Bits%100000000 == 0 && f32Bits > 0 {
			progress := float64(f32Bits) / float64(uint64(1)<<32) * 100
			t.Logf("Progress: %.1f%% (%d / 4294967296)", progress, f32Bits)
		}

		// Check for overflow/completion
		if f32Bits == 0xFFFFFFFF {
			break
		}
		f32Bits++
	}

	if failCount > 0 {
		if failCount > maxFailuresToReport {
			t.Errorf("... and %d more failures (total %d failures out of 4,294,967,296 tests)",
				failCount-maxFailuresToReport, failCount)
		}
		t.Fatalf("Exhaustive Float32 to BFloat16 test failed with %d errors", failCount)
	}
}
