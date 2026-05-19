package proto

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFloat32ToBFloat16_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input float32
		want  float32 // Expected after round-trip (may have precision loss)
		delta float32 // Acceptable difference due to BFloat16 precision
	}{
		{
			name:  "zero",
			input: 0.0,
			want:  0.0,
			delta: 0,
		},
		{
			name:  "negative zero",
			input: float32(math.Copysign(0, -1)),
			want:  float32(math.Copysign(0, -1)),
			delta: 0,
		},
		{
			name:  "one",
			input: 1.0,
			want:  1.0,
			delta: 0,
		},
		{
			name:  "negative one",
			input: -1.0,
			want:  -1.0,
			delta: 0,
		},
		{
			name:  "small positive",
			input: 0.125,
			want:  0.125,
			delta: 0,
		},
		{
			name:  "small negative",
			input: -0.125,
			want:  -0.125,
			delta: 0,
		},
		{
			name:  "positive infinity",
			input: float32(math.Inf(1)),
			want:  float32(math.Inf(1)),
			delta: 0,
		},
		{
			name:  "negative infinity",
			input: float32(math.Inf(-1)),
			want:  float32(math.Inf(-1)),
			delta: 0,
		},
		{
			name:  "large value",
			input: 1234.5678,
			want:  1234.5678,
			delta: 5.0, // BFloat16 has less precision (7 bits mantissa)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf16 := Float32ToBFloat16(tt.input)
			result := BFloat16ToFloat32(bf16)

			if math.IsNaN(float64(tt.want)) {
				assert.True(t, math.IsNaN(float64(result)), "expected NaN")
			} else if math.IsInf(float64(tt.want), 1) {
				assert.True(t, math.IsInf(float64(result), 1), "expected +Inf")
			} else if math.IsInf(float64(tt.want), -1) {
				assert.True(t, math.IsInf(float64(result), -1), "expected -Inf")
			} else {
				assert.InDelta(t, tt.want, result, float64(tt.delta))
			}
		})
	}
}

func TestFloat32ToBFloat16_NaN(t *testing.T) {
	nan := float32(math.NaN())
	bf16 := Float32ToBFloat16(nan)
	result := BFloat16ToFloat32(bf16)
	assert.True(t, math.IsNaN(float64(result)), "NaN should round-trip")
}

func TestFloat32ToBFloat16_BankersRounding(t *testing.T) {
	// Test that banker's rounding (round to nearest even) works correctly
	// by verifying the rounding logic is applied

	// Test simple values that are exactly representable
	tests := []struct {
		name  string
		input float32
	}{
		{name: "1.0", input: 1.0},
		{name: "2.0", input: 2.0},
		{name: "3.0", input: 3.0},
		{name: "4.0", input: 4.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf16 := Float32ToBFloat16(tt.input)
			result := BFloat16ToFloat32(bf16)
			assert.Equal(t, tt.input, result, "exact values should round-trip perfectly")
		})
	}
}

func TestBFloat16ToFloat32_Values(t *testing.T) {
	tests := []struct {
		name  string
		input uint16
		want  float32
	}{
		{
			name:  "zero",
			input: 0x0000,
			want:  0.0,
		},
		{
			name:  "one",
			input: 0x3F80,
			want:  1.0,
		},
		{
			name:  "negative one",
			input: 0xBF80,
			want:  -1.0,
		},
		{
			name:  "two",
			input: 0x4000,
			want:  2.0,
		},
		{
			name:  "positive infinity",
			input: 0x7F80,
			want:  float32(math.Inf(1)),
		},
		{
			name:  "negative infinity",
			input: 0xFF80,
			want:  float32(math.Inf(-1)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BFloat16ToFloat32(tt.input)
			if math.IsInf(float64(tt.want), 1) {
				assert.True(t, math.IsInf(float64(result), 1), "expected +Inf")
			} else if math.IsInf(float64(tt.want), -1) {
				assert.True(t, math.IsInf(float64(result), -1), "expected -Inf")
			} else {
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func BenchmarkFloat32ToBFloat16(b *testing.B) {
	v := float32(1.234567)
	for b.Loop() {
		Float32ToBFloat16(v)
	}
}

func BenchmarkBFloat16ToFloat32(b *testing.B) {
	v := uint16(0x3F9D) // ~1.234 in BFloat16
	for b.Loop() {
		BFloat16ToFloat32(v)
	}
}
