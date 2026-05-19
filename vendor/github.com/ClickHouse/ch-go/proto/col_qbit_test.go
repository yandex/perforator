package proto

import (
	"bytes"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQBitType(t *testing.T) {
	tests := []struct {
		name          string
		input         ColumnType
		wantElement   ColumnType
		wantDimension int
		wantErr       bool
	}{
		{
			name:          "Float32 valid",
			input:         "QBit(Float32, 1024)",
			wantElement:   ColumnTypeFloat32,
			wantDimension: 1024,
			wantErr:       false,
		},
		{
			name:          "Float64 valid",
			input:         "QBit(Float64, 512)",
			wantElement:   ColumnTypeFloat64,
			wantDimension: 512,
			wantErr:       false,
		},
		{
			name:          "BFloat16 valid",
			input:         "QBit(BFloat16, 256)",
			wantElement:   ColumnTypeBFloat16,
			wantDimension: 256,
			wantErr:       false,
		},
		{
			name:    "Invalid element type",
			input:   "QBit(Int32, 128)",
			wantErr: true,
		},
		{
			name:    "Invalid format - missing dimension",
			input:   "QBit(Float32)",
			wantErr: true,
		},
		{
			name:    "Invalid format - too many params",
			input:   "QBit(Float32, 128, 64)",
			wantErr: true,
		},
		{
			name:    "Invalid dimension - negative",
			input:   "QBit(Float32, -1)",
			wantErr: true,
		},
		{
			name:    "Not a QBit type",
			input:   "Float32",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elementType, dimension, err := ParseQBitType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantElement, elementType)
				assert.Equal(t, tt.wantDimension, dimension)
			}
		})
	}
}

func TestColQBit_NewColQBit(t *testing.T) {
	tests := []struct {
		name         string
		elementType  ColumnType
		dimension    int
		wantBitWidth int
		wantErr      bool
	}{
		{
			name:         "BFloat16",
			elementType:  ColumnTypeBFloat16,
			dimension:    128,
			wantBitWidth: 16,
			wantErr:      false,
		},
		{
			name:         "Float32",
			elementType:  ColumnTypeFloat32,
			dimension:    256,
			wantBitWidth: 32,
			wantErr:      false,
		},
		{
			name:         "Float64",
			elementType:  ColumnTypeFloat64,
			dimension:    512,
			wantBitWidth: 64,
			wantErr:      false,
		},
		{
			name:        "Invalid element type",
			elementType: ColumnTypeInt32,
			dimension:   128,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col, err := NewColQBit(tt.elementType, tt.dimension)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.elementType, col.elementType)
				assert.Equal(t, tt.dimension, col.dimension)
				assert.Equal(t, tt.wantBitWidth, col.bitWidth)
				assert.Equal(t, (tt.dimension+7)/8, col.bytesPerRow)
				assert.Equal(t, 0, col.Rows())
			}
		})
	}
}

func TestColQBit_AppendAndRow_Float32(t *testing.T) {
	col, err := NewColQBit(ColumnTypeFloat32, 4)
	require.NoError(t, err)

	// Test vector with known bit patterns
	vector1 := []float32{1.0, 2.0, 3.0, 4.0}
	err = col.Append(vector1)
	require.NoError(t, err)
	assert.Equal(t, 1, col.Rows())

	// Read back the vector
	result := col.Row(0)
	require.NotNil(t, result)
	assert.Equal(t, vector1, result)

	// Add another vector
	vector2 := []float32{5.0, 6.0, 7.0, 8.0}
	err = col.Append(vector2)
	require.NoError(t, err)
	assert.Equal(t, 2, col.Rows())

	// Verify both vectors
	result0 := col.Row(0)
	result1 := col.Row(1)
	assert.Equal(t, vector1, result0)
	assert.Equal(t, vector2, result1)
}

func TestColQBit_AppendAndRow_BFloat16(t *testing.T) {
	col, err := NewColQBit(ColumnTypeBFloat16, 3)
	require.NoError(t, err)

	// BFloat16 has reduced precision
	vector := []float32{1.5, 2.25, 3.125}
	err = col.Append(vector)
	require.NoError(t, err)

	result := col.Row(0)
	require.NotNil(t, result)

	// BFloat16 precision check - should be close but may have rounding
	for i := range vector {
		// BFloat16 has ~3-4 decimal digits of precision
		assert.InDelta(t, vector[i], result[i], 0.01, "element %d", i)
	}
}

func TestColQBit_AppendAndRow_Float64(t *testing.T) {
	col, err := NewColQBit(ColumnTypeFloat64, 2)
	require.NoError(t, err)

	// Test with Float64 values (will be converted to float32 for storage)
	vector := []float32{1.123456789, 2.987654321}
	err = col.Append(vector)
	require.NoError(t, err)

	result := col.Row(0)
	require.NotNil(t, result)

	// Float32 precision check
	for i := range vector {
		assert.InDelta(t, vector[i], result[i], 1e-6, "element %d", i)
	}
}

func TestColQBit_AppendDimensionMismatch(t *testing.T) {
	col, err := NewColQBit(ColumnTypeFloat32, 4)
	require.NoError(t, err)

	// Try to append vector with wrong dimension
	wrongVector := []float32{1.0, 2.0, 3.0} // Only 3 elements
	err = col.Append(wrongVector)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dimension mismatch")
}

func TestColQBit_EncodeDecodeRoundtrip_Float32(t *testing.T) {
	col1, err := NewColQBit(ColumnTypeFloat32, 8)
	require.NoError(t, err)

	// Add multiple vectors
	vectors := [][]float32{
		{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0},
		{10.0, 20.0, 30.0, 40.0, 50.0, 60.0, 70.0, 80.0},
		{-1.5, -2.5, -3.5, -4.5, -5.5, -6.5, -7.5, -8.5},
	}

	for _, vec := range vectors {
		err = col1.Append(vec)
		require.NoError(t, err)
	}

	// Encode
	var buf Buffer
	col1.EncodeColumn(&buf)

	// Decode
	col2, err := NewColQBit(ColumnTypeFloat32, 8)
	require.NoError(t, err)

	reader := NewReader(bytes.NewReader(buf.Buf))
	err = col2.DecodeColumn(reader, 3)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, 3, col2.Rows())
	for i := range 3 {
		result := col2.Row(i)
		assert.Equal(t, vectors[i], result, "vector %d", i)
	}
}

func TestColQBit_EncodeDecodeRoundtrip_BFloat16(t *testing.T) {
	col1, err := NewColQBit(ColumnTypeBFloat16, 4)
	require.NoError(t, err)

	vectors := [][]float32{
		{1.0, 2.0, 3.0, 4.0},
		{10.5, 20.5, 30.5, 40.5},
	}

	for _, vec := range vectors {
		err = col1.Append(vec)
		require.NoError(t, err)
	}

	// Encode
	var buf Buffer
	col1.EncodeColumn(&buf)

	// Decode
	col2, err := NewColQBit(ColumnTypeBFloat16, 4)
	require.NoError(t, err)

	reader := NewReader(bytes.NewReader(buf.Buf))
	err = col2.DecodeColumn(reader, 2)
	require.NoError(t, err)

	// Verify with BFloat16 precision
	assert.Equal(t, 2, col2.Rows())
	for i := range 2 {
		result := col2.Row(i)
		for j := range vectors[i] {
			assert.InDelta(t, vectors[i][j], result[j], 0.01, "vector %d, element %d", i, j)
		}
	}
}

func TestColQBit_Reset(t *testing.T) {
	col, err := NewColQBit(ColumnTypeFloat32, 4)
	require.NoError(t, err)

	// Add some vectors
	err = col.Append([]float32{1.0, 2.0, 3.0, 4.0})
	require.NoError(t, err)
	err = col.Append([]float32{5.0, 6.0, 7.0, 8.0})
	require.NoError(t, err)
	assert.Equal(t, 2, col.Rows())

	// Reset
	col.Reset()
	assert.Equal(t, 0, col.Rows())

	// Verify bit planes are cleared
	for i := range col.bitPlanes {
		assert.Equal(t, 0, len(col.bitPlanes[i]))
	}

	// Can append after reset
	err = col.Append([]float32{10.0, 20.0, 30.0, 40.0})
	require.NoError(t, err)
	assert.Equal(t, 1, col.Rows())
}

func TestColQBit_Type(t *testing.T) {
	tests := []struct {
		elementType ColumnType
		dimension   int
		want        ColumnType
	}{
		{ColumnTypeFloat32, 1024, "QBit(Float32, 1024)"},
		{ColumnTypeBFloat16, 256, "QBit(BFloat16, 256)"},
		{ColumnTypeFloat64, 512, "QBit(Float64, 512)"},
	}

	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			col, err := NewColQBit(tt.elementType, tt.dimension)
			require.NoError(t, err)
			assert.Equal(t, tt.want, col.Type())
		})
	}
}

func TestColQBit_Infer(t *testing.T) {
	tests := []struct {
		name         string
		colType      ColumnType
		wantElement  ColumnType
		wantDim      int
		wantBitWidth int
		wantErr      bool
	}{
		{
			name:         "Float32 1024",
			colType:      "QBit(Float32, 1024)",
			wantElement:  ColumnTypeFloat32,
			wantDim:      1024,
			wantBitWidth: 32,
			wantErr:      false,
		},
		{
			name:         "BFloat16 128",
			colType:      "QBit(BFloat16, 128)",
			wantElement:  ColumnTypeBFloat16,
			wantDim:      128,
			wantBitWidth: 16,
			wantErr:      false,
		},
		{
			name:    "Invalid type",
			colType: "Float32",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := &ColQBit{}
			err := col.Infer(tt.colType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantElement, col.elementType)
				assert.Equal(t, tt.wantDim, col.dimension)
				assert.Equal(t, tt.wantBitWidth, col.bitWidth)
				assert.Equal(t, 0, col.Rows())
			}
		})
	}
}

func TestColQBit_SpecialValues_Float32(t *testing.T) {
	col, err := NewColQBit(ColumnTypeFloat32, 6)
	require.NoError(t, err)

	// In Go, both 0.0 and -0.0 are exactly same if used as constants.
	// Hence using it as variable to have sign bit different between those two.
	pZero := float32(0.0)
	nZero := -1 * pZero

	// Test special float values
	vector := []float32{
		pZero,
		nZero,
		float32(math.Inf(1)),  // +Inf
		float32(math.Inf(-1)), // -Inf
		float32(math.NaN()),   // NaN
		1.0,
	}

	err = col.Append(vector)
	require.NoError(t, err)

	result := col.Row(0)
	require.NotNil(t, result)

	assert.Equal(t, nZero, result[0])
	assert.Equal(t, pZero, result[1])
	assert.True(t, math.IsInf(float64(result[2]), 1), "should be +Inf")
	assert.True(t, math.IsInf(float64(result[3]), -1), "should be -Inf")
	assert.True(t, math.IsNaN(float64(result[4])), "should be NaN")
	assert.Equal(t, float32(1.0), result[5])
}

func TestColQBit_LargeDimension(t *testing.T) {
	// Test with a larger dimension (like real embeddings)
	dimension := 1536 // Common OpenAI embedding dimension
	col, err := NewColQBit(ColumnTypeFloat32, dimension)
	require.NoError(t, err)

	// Create a test vector
	vector := make([]float32, dimension)
	for i := range vector {
		vector[i] = float32(i) / float32(dimension)
	}

	err = col.Append(vector)
	require.NoError(t, err)

	result := col.Row(0)
	require.NotNil(t, result)
	assert.Equal(t, dimension, len(result))

	// Verify a few sample values
	assert.Equal(t, vector[0], result[0])
	assert.Equal(t, vector[dimension/2], result[dimension/2])
	assert.Equal(t, vector[dimension-1], result[dimension-1])
}

func TestColQBit_MultipleVectorsIntegrity(t *testing.T) {
	// Test that multiple vectors don't interfere with each other
	col, err := NewColQBit(ColumnTypeFloat32, 16)
	require.NoError(t, err)

	numVectors := 100
	vectors := make([][]float32, numVectors)

	// Create and append vectors
	for i := range numVectors {
		vectors[i] = make([]float32, 16)
		for j := range 16 {
			vectors[i][j] = float32(i*16 + j)
		}
		err = col.Append(vectors[i])
		require.NoError(t, err)
	}

	assert.Equal(t, numVectors, col.Rows())

	// Verify all vectors
	for i := range numVectors {
		result := col.Row(i)
		assert.Equal(t, vectors[i], result, "vector %d", i)
	}
}

func TestColQBit_RowOutOfBounds(t *testing.T) {
	col, err := NewColQBit(ColumnTypeFloat32, 4)
	require.NoError(t, err)

	err = col.Append([]float32{1.0, 2.0, 3.0, 4.0})
	require.NoError(t, err)

	// Test negative index
	result := col.Row(-1)
	assert.Nil(t, result)

	// Test index >= rows
	result = col.Row(1)
	assert.Nil(t, result)

	result = col.Row(100)
	assert.Nil(t, result)
}

// Benchmark tests
func BenchmarkColQBit_Append_Float32_1536(b *testing.B) {
	col, _ := NewColQBit(ColumnTypeFloat32, 1536)
	vector := make([]float32, 1536)
	for i := range vector {
		vector[i] = float32(i)
	}

	for b.Loop() {
		col.Append(vector)
	}
}

func BenchmarkColQBit_Row_Float32_1536(b *testing.B) {
	col, _ := NewColQBit(ColumnTypeFloat32, 1536)
	vector := make([]float32, 1536)
	for i := range vector {
		vector[i] = float32(i)
	}
	col.Append(vector)

	for b.Loop() {
		col.Row(0)
	}
}

func BenchmarkColQBit_EncodeColumn_Float32_1536_1000rows(b *testing.B) {
	col, _ := NewColQBit(ColumnTypeFloat32, 1536)
	vector := make([]float32, 1536)
	for i := range vector {
		vector[i] = float32(i)
	}
	for range 1000 {
		col.Append(vector)
	}

	for b.Loop() {
		var buf Buffer
		col.EncodeColumn(&buf)
	}
}
