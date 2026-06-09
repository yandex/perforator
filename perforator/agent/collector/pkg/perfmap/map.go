package perfmap

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/yandex/perforator/perforator/internal/symboltable"
	"github.com/yandex/perforator/perforator/pkg/disjointsegmentsets"
)

type perfMap struct {
	path             string
	lastRefreshMtime time.Time
	lastRefreshSize  int64
	symCount         int
}

func newPerfMap(path string) *perfMap {
	pm := &perfMap{
		path: path,
	}

	return pm
}

type refreshStats struct {
	skipped     bool
	rebuildTime time.Duration
	currentSize int
}

func (p *perfMap) refresh() ([]symboltable.Entry[string], refreshStats, error) {
	info, err := os.Stat(p.path)
	if err != nil {
		return nil, refreshStats{}, fmt.Errorf("failed to stat perf map: %w", err)
	}
	var stats refreshStats
	if p.lastRefreshSize == info.Size() && p.lastRefreshMtime.Equal(info.ModTime()) {
		stats.skipped = true
		stats.currentSize = p.symCount
		return nil, stats, nil
	}

	file, err := os.Open(p.path)
	if err != nil {
		return nil, stats, fmt.Errorf("failed to open perf map: %w", err)
	}
	defer file.Close()

	startTS := time.Now()
	rawSyms, err := parse(file)
	if err != nil {
		return nil, stats, fmt.Errorf("failed to parse perf map: %w", err)
	}
	slices.SortFunc(rawSyms, func(a, b symbol) int {
		return cmp.Compare(a.offset, b.offset)
	})
	rawSyms, _ = disjointsegmentsets.Prune(rawSyms)
	stats.rebuildTime = time.Since(startTS)
	p.symCount = len(rawSyms)
	stats.currentSize = p.symCount

	p.lastRefreshMtime = info.ModTime()
	p.lastRefreshSize = info.Size()
	var syms []symboltable.Entry[string]
	for _, s := range rawSyms {
		syms = append(syms, symboltable.Entry[string]{
			Data:  s.name,
			Begin: s.offset,
			Size:  s.size,
		})
	}
	return syms, stats, nil
}
