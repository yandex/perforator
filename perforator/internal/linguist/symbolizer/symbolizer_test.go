package symbolizer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/internal/unwinder"
)

func TestExtractNameAndFilenameSlices(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width         uint8
		functionBytes []byte
		filenameBytes []byte
	}{
		{
			name:          "production ASCII overflow",
			width:         1,
			functionBytes: []byte("<module>"),
			filenameBytes: []byte(strings.Repeat("f", 255)),
		},
		{
			name:          "maximum ASCII lengths",
			width:         1,
			functionBytes: bytes.Repeat([]byte{'n'}, 255),
			filenameBytes: bytes.Repeat([]byte{'f'}, 255),
		},
		{
			name:          "UTF-16 offsets",
			width:         2,
			functionBytes: []byte{0x44, 0x04, 0x00, 0x00},
			filenameBytes: []byte{0x44, 0x04, '.', 0x00, 'p', 0x00, 'y', 0x00},
		},
		{
			name:          "UTF-32 offsets",
			width:         4,
			functionBytes: []byte{0x00, 0xf6, 0x01, 0x00},
			filenameBytes: []byte{'f', 0x00, 0x00, 0x00, '.', 0x00, 0x00, 0x00},
		},
		{
			name:          "UTF-32 fills buffer exactly",
			width:         4,
			functionBytes: bytes.Repeat([]byte{'n', 0, 0, 0}, 255),
			filenameBytes: []byte{'f', 0, 0, 0},
		},
		{
			name:  "empty strings",
			width: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			symbol := unwinder.Symbol{
				NameLength:     uint8(len(tc.functionBytes) / int(tc.width)),
				FilenameLength: uint8(len(tc.filenameBytes) / int(tc.width)),
				CodepointSize:  tc.width,
			}
			copy(symbol.Data[:], tc.functionBytes)
			copy(symbol.Data[len(tc.functionBytes):], tc.filenameBytes)

			name, filename, ok := extractNameAndFilenameSlices(&symbol)
			require.True(t, ok)
			require.True(t, bytes.Equal(tc.functionBytes, name), "function bytes differ")
			require.True(t, bytes.Equal(tc.filenameBytes, filename), "filename bytes differ")
		})
	}
}

func TestExtractNameAndFilenameSlicesRejectsInvalidMetadata(t *testing.T) {
	for _, tc := range []struct {
		name   string
		symbol unwinder.Symbol
	}{
		{
			name:   "UTF-32 exceeds buffer by one codepoint",
			symbol: unwinder.Symbol{NameLength: 255, FilenameLength: 2, CodepointSize: 4},
		},
		{
			name:   "maximum UTF-32 lengths exceed buffer",
			symbol: unwinder.Symbol{NameLength: 255, FilenameLength: 255, CodepointSize: 4},
		},
		{
			name:   "zero codepoint size",
			symbol: unwinder.Symbol{},
		},
		{
			name:   "unsupported codepoint size within buffer",
			symbol: unwinder.Symbol{NameLength: 1, FilenameLength: 1, CodepointSize: 3},
		},
		{
			name:   "unsupported codepoint size exceeds buffer",
			symbol: unwinder.Symbol{NameLength: 255, FilenameLength: 255, CodepointSize: 255},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name, filename, ok := extractNameAndFilenameSlices(&tc.symbol)
			require.False(t, ok)
			require.Nil(t, name)
			require.Nil(t, filename)
		})
	}
}

func TestExtractNameAndFilenameSlicesAllLengths(t *testing.T) {
	var symbol unwinder.Symbol
	for i := range symbol.Data {
		symbol.Data[i] = byte(i)
	}
	for _, width := range []uint8{1, 2, 4} {
		symbol.CodepointSize = width
		for n := 0; n <= 255; n++ {
			symbol.NameLength = uint8(n)
			for f := 0; f <= 255; f++ {
				symbol.FilenameLength = uint8(f)
				name, filename, ok := extractNameAndFilenameSlices(&symbol)
				wantOK := (n+f)*int(width) <= len(symbol.Data)
				if ok != wantOK {
					t.Fatalf("width=%d name=%d filename=%d: accepted=%v, want %v", width, n, f, ok, wantOK)
				}
				if !ok {
					if name != nil || filename != nil {
						t.Fatal("invalid symbol returned non-nil slices")
					}
					continue
				}
				nameEnd := n * int(width)
				fileEnd := (n + f) * int(width)
				if !bytes.Equal(name, symbol.Data[:nameEnd]) || !bytes.Equal(filename, symbol.Data[nameEnd:fileEnd]) {
					t.Fatalf("width=%d name=%d filename=%d: incorrect slices", width, n, f)
				}
			}
		}
	}
}
