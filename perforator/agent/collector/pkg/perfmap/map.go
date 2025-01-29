package perfmap

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"sync/atomic"
	"time"

	"github.com/yandex/perforator/perforator/pkg/disjointsegmentsets"
)

type perfMap struct {
	path             string
	lastRefreshMtime time.Time
	lastRefreshSize  int64
	symbols          atomic.Pointer[[]symbol]
}

func newPerfMap(path string) *perfMap {
	pm := &perfMap{
		path: path,
	}

	pm.symbols.Store(&[]symbol{})

	return pm
}

// can be called concurrently without restrictions
func (p *perfMap) find(ip uint64) (string, bool) {
	symbols := *p.symbols.Load()
	pos, ok := slices.BinarySearchFunc(symbols, 42, func(s symbol, unused int) int {
		if s.offset <= ip {
			if ip < s.offset+s.size {
				return 0
			}
			return -1
		}
		return 1
	})
	if !ok {
		return "", false
	}
	return symbols[pos].name, true
}

type refreshStats struct {
	skipped     bool
	rebuildTime time.Duration
	currentSize int
}

func (p *perfMap) refresh() (refreshStats, error) {
	info, err := os.Stat(p.path)
	if err != nil {
		return refreshStats{}, fmt.Errorf("failed to stat perf map: %w", err)
	}
	var stats refreshStats
	if p.lastRefreshSize == info.Size() && p.lastRefreshMtime.Equal(info.ModTime()) {
		stats.skipped = true
		stats.currentSize = len(*p.symbols.Load())
		return stats, nil
	}

	file, err := os.Open(p.path)
	if err != nil {
		return stats, fmt.Errorf("failed to open perf map: %w", err)
	}
	defer file.Close()

	startTS := time.Now()
	syms, err := parse(file)
	if err != nil {
		return stats, fmt.Errorf("failed to parse perf map: %w", err)
	}
	slices.SortFunc(syms, func(a, b symbol) int {
		return cmp.Compare(a.offset, b.offset)
	})
	syms, _ = disjointsegmentsets.Prune(syms)
	stats.rebuildTime = time.Since(startTS)
	stats.currentSize = len(syms)

	p.lastRefreshMtime = info.ModTime()
	p.lastRefreshSize = info.Size()

	p.symbols.Store(&syms)
	return stats, nil
}
