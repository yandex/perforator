package symbolizer

import (
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"

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
	cacheMisses   metrics.Counter
	cacheHits     metrics.Counter
	cacheSizeFunc metrics.FuncIntGauge
	cacheCapacity metrics.IntGauge
}

type Symbol struct {
	FileName string
	QualName string
}

type Symbolizer struct {
	reg   metrics.Registry
	c     *SymbolizerConfig
	bpf   *machine.BPF
	cache *lru.Cache[unwinder.PythonSymbolKey, *Symbol]

	metrics *symbolizerMetrics
}

func NewSymbolizer(c *SymbolizerConfig, bpf *machine.BPF, reg metrics.Registry) (*Symbolizer, error) {
	cacheSize := DefaultMaxCacheSize
	if c.MaxCacheSize != 0 {
		cacheSize = int(c.MaxCacheSize)
	}

	cache, err := lru.New[unwinder.PythonSymbolKey, *Symbol](cacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create lru cache: %v", err)
	}

	res := &Symbolizer{
		reg:   reg,
		c:     c,
		bpf:   bpf,
		cache: cache,
		metrics: &symbolizerMetrics{
			cacheMisses: reg.WithTags(map[string]string{"type": "miss"}).Counter("python.symbolize.cache.access.count"),
			cacheHits:   reg.WithTags(map[string]string{"type": "hit"}).Counter("python.symbolize.cache.access.count"),
			cacheSizeFunc: reg.FuncIntGauge("python.symbolize.cache.size", func() int64 {
				return int64(cache.Len())
			}),
			cacheCapacity: reg.IntGauge("python.symbolize.cache.capacity"),
		},
	}
	res.metrics.cacheCapacity.Set(int64(cacheSize))

	return res, nil
}

func (s *Symbolizer) Symbolize(key *unwinder.PythonSymbolKey) (*Symbol, bool) {
	if symbol, ok := s.cache.Get(*key); ok {
		s.metrics.cacheHits.Inc()
		return symbol, true
	}

	s.metrics.cacheMisses.Inc()

	symbol, exists := s.bpf.SymbolizePython(key)
	if !exists {
		return nil, false
	}

	newSymbol := &Symbol{
		FileName: copy.ZeroTerminatedString(symbol.FileName[:]),
		QualName: copy.ZeroTerminatedString(symbol.QualName[:]),
	}
	_ = s.cache.Add(*key, newSymbol)

	return newSymbol, true
}
