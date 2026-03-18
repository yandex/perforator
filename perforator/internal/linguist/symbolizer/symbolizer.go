package symbolizer

import (
	"bytes"
	"encoding/binary"
	"time"
	"unsafe"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/copy"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/programstate"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

const (
	DefaultMaxCacheSize = (1 << 13)
)

type SymbolizerConfig struct {
	MaxCacheSize uint64        `yaml:"max_cache_size"`
	ItemTTL      time.Duration `yaml:"item_ttl"`
}

type symbolizerMetrics struct {
	cacheMisses     metrics.Counter
	cacheHits       metrics.Counter
	cacheSizeFunc   metrics.FuncIntGauge
	cacheCapacity   metrics.IntGauge
	failedDecodeUTF metrics.Counter
}

type Symbol struct {
	FileName string
	Name     string
}

type Symbolizer struct {
	reg   metrics.Registry
	c     *SymbolizerConfig
	state *programstate.State
	cache *expirable.LRU[unwinder.SymbolKey, *Symbol]

	metrics *symbolizerMetrics
}

func NewPythonSymbolizer(c *SymbolizerConfig, state *programstate.State, reg metrics.Registry) (*Symbolizer, error) {
	return newSymbolizer(c, state, reg, "python")
}

func NewPhpSymbolizer(c *SymbolizerConfig, state *programstate.State, reg metrics.Registry) (*Symbolizer, error) {
	return newSymbolizer(c, state, reg, "php")
}

func newSymbolizer(c *SymbolizerConfig, state *programstate.State, reg metrics.Registry, language string) (*Symbolizer, error) {
	cacheSize := DefaultMaxCacheSize
	itemTTL := 5 * time.Minute
	if c.ItemTTL != 0 {
		itemTTL = c.ItemTTL
	}
	if c.MaxCacheSize != 0 {
		cacheSize = int(c.MaxCacheSize)
	}

	cache := expirable.NewLRU[unwinder.SymbolKey, *Symbol](cacheSize, nil, itemTTL)

	res := &Symbolizer{
		reg:   reg,
		c:     c,
		state: state,
		cache: cache,
		metrics: &symbolizerMetrics{
			cacheMisses: reg.WithTags(map[string]string{"type": "miss"}).Counter(language + ".symbolize.cache.access.count"),
			cacheHits:   reg.WithTags(map[string]string{"type": "hit"}).Counter(language + ".symbolize.cache.access.count"),
			cacheSizeFunc: reg.FuncIntGauge(language+".symbolize.cache.size", func() int64 {
				return int64(cache.Len())
			}),
			cacheCapacity:   reg.IntGauge(language + ".symbolize.cache.capacity"),
			failedDecodeUTF: reg.Counter(language + ".symbolize.failed_decode_utf"),
		},
	}
	res.metrics.cacheCapacity.Set(int64(cacheSize))

	return res, nil
}

func bytesToIntSlice[T uint16 | uint32](data []byte) []T {
	if len(data) == 0 {
		return nil
	}

	elemSize := int(unsafe.Sizeof(T(0)))

	count := len(data) / elemSize
	result := make([]T, count)

	reader := bytes.NewReader(data[0 : count*elemSize])

	err := binary.Read(reader, binary.LittleEndian, &result)
	if err != nil {
		return nil
	}

	return result
}

func extractNullTerminated[T uint16 | uint32](data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	intSlice := bytesToIntSlice[T](data)

	for i, val := range intSlice {
		if val == 0 {
			return data[:i*int(unsafe.Sizeof(T(0)))]
		}
	}

	return data
}

func (s *Symbolizer) decodeUTF16(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	validBytes := extractNullTerminated[uint16](data)
	if len(validBytes) == 0 {
		return ""
	}

	decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
	result, err := decoder.Bytes(validBytes)
	if err != nil {
		s.metrics.failedDecodeUTF.Inc()
		return ""
	}

	return string(result)
}

func (s *Symbolizer) decodeUTF32(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	validBytes := extractNullTerminated[uint32](data)
	if len(validBytes) == 0 {
		return ""
	}

	decoder := utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewDecoder()
	result, err := decoder.Bytes(validBytes)
	if err != nil {
		s.metrics.failedDecodeUTF.Inc()
		return ""
	}

	return string(result)
}

func extractNameAndFilenameSlices(symbol *unwinder.Symbol) (nameBytes, filenameBytes []byte) {
	if symbol.CodepointSize == 1 {
		nameBytes = symbol.Data[:symbol.NameLength]
		filenameBytes = symbol.Data[symbol.NameLength : symbol.NameLength+symbol.FilenameLength]
		return
	}

	// UTF-16 or UTF-32
	charSize := int(symbol.CodepointSize)
	filenameOffset := int(symbol.NameLength) * charSize
	nameBytes = symbol.Data[:filenameOffset]
	filenameBytes = symbol.Data[filenameOffset : filenameOffset+int(symbol.FilenameLength)*charSize]
	return
}

// decodeSymbol converts a raw eBPF symbol into a Go Symbol.
// Returns nil for unknown codepoint sizes.
func (s *Symbolizer) decodeSymbol(raw *unwinder.Symbol) *Symbol {
	nameBytes, filenameBytes := extractNameAndFilenameSlices(raw)
	var name, fileName string
	switch raw.CodepointSize {
	case 1:
		name = copy.ZeroTerminatedString(nameBytes)
		fileName = copy.ZeroTerminatedString(filenameBytes)
	case 2:
		name = s.decodeUTF16(nameBytes)
		fileName = s.decodeUTF16(filenameBytes)
	case 4:
		name = s.decodeUTF32(nameBytes)
		fileName = s.decodeUTF32(filenameBytes)
	default:
		return nil
	}
	return &Symbol{Name: name, FileName: fileName}
}

func (s *Symbolizer) Symbolize(key *unwinder.SymbolKey) (*Symbol, bool) {
	if symbol, ok := s.cache.Get(*key); ok {
		s.metrics.cacheHits.Inc()
		return symbol, true
	}

	s.metrics.cacheMisses.Inc()

	raw, exists := s.state.SymbolizeInterpeter(key)
	if !exists {
		return nil, false
	}

	sym := s.decodeSymbol(&raw)
	if sym == nil {
		return nil, false
	}

	_ = s.cache.Add(*key, sym)
	return sym, true
}

// SymbolizeBatch resolves all keys in two passes: LRU cache hits first,
// then a single sequential eBPF map scan for cache misses.
// results[i] == nil iff key[i] is absent from both cache and eBPF map.
func (s *Symbolizer) SymbolizeBatch(keys []unwinder.SymbolKey) []*Symbol {
	results := make([]*Symbol, len(keys))

	// Pass 1: serve hits from LRU cache.
	missIdx := make([]int, 0, len(keys))
	for i, key := range keys {
		if sym, ok := s.cache.Get(key); ok {
			s.metrics.cacheHits.Inc()
			results[i] = sym
		} else {
			s.metrics.cacheMisses.Inc()
			missIdx = append(missIdx, i)
		}
	}
	if len(missIdx) == 0 {
		return results
	}

	// Pass 2: batch-fetch cache misses from eBPF map.
	missKeys := make([]unwinder.SymbolKey, len(missIdx))
	for j, idx := range missIdx {
		missKeys[j] = keys[idx]
	}
	rawSymbols, found := s.state.SymbolizeInterpreterBatch(missKeys)
	for j, idx := range missIdx {
		if !found[j] {
			continue
		}
		sym := s.decodeSymbol(&rawSymbols[j])
		if sym == nil {
			continue
		}
		_ = s.cache.Add(missKeys[j], sym)
		results[idx] = sym
	}
	return results
}
