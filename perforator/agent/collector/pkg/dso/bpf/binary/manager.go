package binary

import (
	"sync"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	pythonbpf "github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/python"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/tls"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/unwindtable"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
)

type Allocation struct {
	BuildID string
	id      uint64

	// Mapping of tls offsets to variable names
	TLSMutex sync.RWMutex
	TLSMap   map[uint64]string

	UnwindTableAllocation *unwindtable.Allocation
}

type BPFBinaryManager struct {
	l log.Logger

	tables *unwindtable.BPFManager
	tls    *tls.BPFManager
	python *pythonbpf.BPFManager
}

func NewBPFBinaryManager(l log.Logger, r metrics.Registry, bpf *machine.BPF) (*BPFBinaryManager, error) {
	l = l.WithName("BinaryManager")

	unwmanager, err := unwindtable.NewBPFManager(l, r, bpf)
	if err != nil {
		return nil, err
	}

	tlsmanager, err := tls.NewBPFManager(l, bpf)
	if err != nil {
		return nil, err
	}

	return &BPFBinaryManager{
		l:      l,
		tables: unwmanager,
		tls:    tlsmanager,
		python: pythonbpf.NewBPFManager(l, bpf),
	}, nil
}

func (m *BPFBinaryManager) Add(buildID string, id uint64, analysis *parse.BinaryAnalysis) (alloc *Allocation, err error) {
	unwAlloc, err := m.tables.Add(buildID, id, analysis.UnwindTable)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			m.tables.Release(unwAlloc)
		}
	}()

	err = m.tls.Add(id, analysis.TLSConfig)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			m.tls.Release(id)
		}
	}()

	err = m.python.Add(id, analysis.PythonConfig)
	if err != nil {
		return nil, err
	}

	alloc = &Allocation{
		BuildID:               buildID,
		id:                    id,
		TLSMap:                map[uint64]string{},
		UnwindTableAllocation: unwAlloc,
	}

	return alloc, err
}

func (m *BPFBinaryManager) Release(a *Allocation) {
	m.tables.Release(a.UnwindTableAllocation)
	m.tls.Release(a.id)
	m.python.Release(a.id)
}

func (m *BPFBinaryManager) MoveFromCache(a *Allocation) bool {
	return m.tables.MoveFromCache(a.UnwindTableAllocation)
}

func (m *BPFBinaryManager) MoveToCache(a *Allocation) bool {
	return m.tables.MoveToCache(a.UnwindTableAllocation)
}
