package symbolizer

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
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

type fakeSymbolSource struct {
	symbols map[unwinder.InterpreterSymbolKey]unwinder.Symbol
	calls   int
}

func (s *fakeSymbolSource) SymbolizeInterpreter(language models.Language, process linux.ProcessKey, key *unwinder.SymbolKey) (unwinder.Symbol, bool) {
	s.calls++
	symbol, ok := s.symbols[unwinder.InterpreterSymbolKey{SymbolKey: *key, Pid: process.Pid, ProcessStarttime: process.ProcessStartTime, Language: uint8(language)}]
	return symbol, ok
}

func makeSymbol(name string) unwinder.Symbol {
	symbol := unwinder.Symbol{CodepointSize: 1, NameLength: uint8(len(name))}
	copy(symbol.Data[:], name)
	return symbol
}

func TestSymbolCacheSeparatesLanguagesAndProcessLifetimes(t *testing.T) {
	source := &fakeSymbolSource{symbols: make(map[unwinder.InterpreterSymbolKey]unwinder.Symbol)}
	s, err := newSymbolizer(&SymbolizerConfig{}, source, nop.Registry{}, "test")
	require.NoError(t, err)
	key := unwinder.SymbolKey{ObjectAddr: 0x1000, Linestart: 10}
	for _, pid := range []uint32{42, 43} {
		for _, language := range []uint8{uint8(unwinder.LanguagePython), uint8(unwinder.LanguagePhp), uint8(unwinder.LanguageLua)} {
			for _, startTime := range []uint64{100, 200} {
				name := fmt.Sprintf("%d/%d/%d", pid, language, startTime)
				source.symbols[unwinder.InterpreterSymbolKey{SymbolKey: key, Pid: pid, Language: language, ProcessStarttime: startTime}] = makeSymbol(name)
			}
		}
	}
	for iteration := 0; iteration < 2; iteration++ {
		for cacheKey := range source.symbols {
			symbol, ok := s.Symbolize(models.Language(cacheKey.Language), linux.ProcessKey{Pid: cacheKey.Pid, ProcessStartTime: cacheKey.ProcessStarttime}, &cacheKey.SymbolKey)
			require.True(t, ok)
			require.Equal(t, fmt.Sprintf("%d/%d/%d", cacheKey.Pid, cacheKey.Language, cacheKey.ProcessStarttime), symbol.Name)
		}
	}
	require.Equal(t, 12, source.calls, "the second pass must use the Go cache")
}

func TestSymbolCacheRetriesMissingSymbols(t *testing.T) {
	source := &fakeSymbolSource{symbols: make(map[unwinder.InterpreterSymbolKey]unwinder.Symbol)}
	s, err := newSymbolizer(&SymbolizerConfig{}, source, nop.Registry{}, "test")
	require.NoError(t, err)
	key := unwinder.InterpreterSymbolKey{SymbolKey: unwinder.SymbolKey{ObjectAddr: 0x1000}, Pid: 42, Language: uint8(unwinder.LanguagePython), ProcessStarttime: 100}
	_, ok := s.Symbolize(models.Language(key.Language), linux.ProcessKey{Pid: key.Pid, ProcessStartTime: key.ProcessStarttime}, &key.SymbolKey)
	require.False(t, ok)
	source.symbols[key] = makeSymbol("available")
	for i := 0; i < 2; i++ {
		symbol, ok := s.Symbolize(models.Language(key.Language), linux.ProcessKey{Pid: key.Pid, ProcessStartTime: key.ProcessStarttime}, &key.SymbolKey)
		require.True(t, ok)
		require.Equal(t, "available", symbol.Name)
	}
	require.Equal(t, 2, source.calls)
}
