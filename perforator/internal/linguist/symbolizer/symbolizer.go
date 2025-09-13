package symbolizer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/copy"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

const (
	DefaultMaxCacheSize = (1 << 13)
)

type SymbolizerConfig struct {
	MaxCacheSize uint64 `yaml:"max_cache_size"`
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
	bpf   *machine.BPF
	cache *lru.Cache[unwinder.SymbolKey, *Symbol]

	metrics *symbolizerMetrics
}

func NewPythonSymbolizer(c *SymbolizerConfig, bpf *machine.BPF, reg metrics.Registry) (*Symbolizer, error) {
	return newSymbolizer(c, bpf, reg, "python")
}

func NewPhpSymbolizer(c *SymbolizerConfig, bpf *machine.BPF, reg metrics.Registry) (*Symbolizer, error) {
	return newSymbolizer(c, bpf, reg, "php")
}

func newSymbolizer(c *SymbolizerConfig, bpf *machine.BPF, reg metrics.Registry, language string) (*Symbolizer, error) {
	cacheSize := DefaultMaxCacheSize
	if c.MaxCacheSize != 0 {
		cacheSize = int(c.MaxCacheSize)
	}

	cache, err := lru.New[unwinder.SymbolKey, *Symbol](cacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create lru cache: %v", err)
	}

	res := &Symbolizer{
		reg:   reg,
		c:     c,
		bpf:   bpf,
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

func (s *Symbolizer) Symbolize(key *unwinder.SymbolKey) (*Symbol, bool) {
	if symbol, ok := s.cache.Get(*key); ok {
		s.metrics.cacheHits.Inc()
		return symbol, true
	}

	s.metrics.cacheMisses.Inc()

	symbol, exists := s.bpf.SymbolizeInterpeter(key)
	if !exists {
		return nil, false
	}

	var name, fileName string
	nameBytes, filenameBytes := extractNameAndFilenameSlices(&symbol)

	switch symbol.CodepointSize {
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
		return nil, false
	}

	newSymbol := &Symbol{
		Name:     name,
		FileName: fileName,
	}

	_ = s.cache.Add(*key, newSymbol)
	return newSymbol, true
}
