package process

import (
	"context"
	"os"
	"strconv"

	"github.com/yandex/perforator/perforator/pkg/linux"
)

type ProcessScanner interface {
	Scan(ctx context.Context, discoverer func(context.Context, linux.ProcessID)) error
}

////////////////////////////////////////////////////////////////////////////////

type ProcFSScanner struct{}

func (p *ProcFSScanner) Scan(ctx context.Context, discoverer func(context.Context, linux.ProcessID)) (err error) {
	procDir, err := os.Open("/proc")
	if err != nil {
		return
	}

	entries, err := procDir.ReadDir(0 /* read all dir entries */)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			pid, err := strconv.ParseUint(entry.Name(), 10, 32)
			if err != nil { // not a pid directory
				continue
			}
			discoverer(ctx, linux.ProcessID(pid))
		}
	}
	return
}

////////////////////////////////////////////////////////////////////////////////

type ProcessFilter func(pid linux.ProcessID) bool

////////////////////////////////////////////////////////////////////////////////

type FilteringProcessScanner struct {
	underlying ProcessScanner
	filter     ProcessFilter
}

type filteringDiscoverer struct {
	underlying func(context.Context, linux.ProcessID)
	filter     ProcessFilter
}

func (d *filteringDiscoverer) Discover(ctx context.Context, pid linux.ProcessID) {
	if d.filter(pid) {
		d.underlying(ctx, pid)
	}
}

func (s *FilteringProcessScanner) Scan(ctx context.Context, discoverer func(context.Context, linux.ProcessID)) (err error) {
	d := &filteringDiscoverer{underlying: discoverer, filter: s.filter}
	return s.underlying.Scan(ctx, d.Discover)
}

func NewFilteringProcessScanner(underlying ProcessScanner, filter ProcessFilter) *FilteringProcessScanner {
	return &FilteringProcessScanner{
		underlying: underlying,
		filter:     filter,
	}
}

////////////////////////////////////////////////////////////////////////////////

// Compile-time inheritance check.
var _ ProcessScanner = &ProcFSScanner{}

// Compile-time inheritance check.
var _ ProcessScanner = &FilteringProcessScanner{}
