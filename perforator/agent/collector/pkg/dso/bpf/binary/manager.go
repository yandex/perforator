package binary

import (
	"sync"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/unwindtable"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/php"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/pthread"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/python"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/tls"
	php_agent "github.com/yandex/perforator/perforator/internal/linguist/php/agent"
	python_agent "github.com/yandex/perforator/perforator/internal/linguist/python/agent"
	"github.com/yandex/perforator/perforator/internal/unwinder"
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
	l      log.Logger
	bpf    *machine.BPF
	tables *unwindtable.BPFManager
}

func NewBPFBinaryManager(l log.Logger, r metrics.Registry, bpf *machine.BPF) (*BPFBinaryManager, error) {
	l = l.WithName("BinaryManager")

	unwmanager, err := unwindtable.NewBPFManager(l, r, bpf)
	if err != nil {
		return nil, err
	}

	return &BPFBinaryManager{
		l:      l,
		bpf:    bpf,
		tables: unwmanager,
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

	binId := unwinder.BinaryId(id)

	if analysis.TLSConfig != nil {
		err = m.bpf.AddTLSConfig(binId, convertToUnwindTLSConfig(analysis.TLSConfig))
		if err != nil {
			return nil, err
		}
		defer func() {
			if err != nil {
				m.releaseTLS(binId)
			}
		}()
	}

	if analysis.PythonConfig != nil {
		err = m.bpf.AddPythonConfig(binId, convertToUnwindPythonConfig(analysis.PythonConfig))
		if err != nil {
			return nil, err
		}
		defer func() {
			if err != nil {
				m.releasePython(binId)
			}
		}()
	}

	if analysis.PhpConfig != nil {
		err = m.bpf.AddPhpConfig(binId, convertToUnwindPhpConfig(analysis.PhpConfig))
		if err != nil {
			return nil, err
		}

		defer func() {
			if err != nil {
				m.releasePhp(binId)
			}
		}()
	}

	if analysis.PthreadConfig != nil {
		err = m.bpf.AddPthreadConfig(binId, convertToUnwindPthreadConfig(analysis.PthreadConfig))
		if err != nil {
			return nil, err
		}
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
	binId := unwinder.BinaryId(a.id)
	m.releasePthread(binId)
	m.releasePython(binId)
	m.releasePhp(binId)
	m.releaseTLS(binId)
}

func (m *BPFBinaryManager) releasePython(id unwinder.BinaryId) {
	err := m.bpf.DeletePythonConfig(id)
	if err != nil {
		m.l.Error("Failed to delete python config", log.Error(err))
	}
}

func (m *BPFBinaryManager) releasePthread(id unwinder.BinaryId) {
	err := m.bpf.DeletePthreadConfig(id)
	if err != nil {
		m.l.Error("Failed to delete pthread config", log.Error(err))
	}
}

func (m *BPFBinaryManager) releasePhp(id unwinder.BinaryId) {
	err := m.bpf.DeletePhpConfig(id)
	if err != nil {
		m.l.Error("Failed to delete php config", log.Error(err))
	}
}

func (m *BPFBinaryManager) releaseTLS(id unwinder.BinaryId) {
	err := m.bpf.DeleteTLSConfig(id)
	if err != nil {
		m.l.Error("Failed to delete TLS config", log.Error(err))
	}
}

func convertToUnwindTLSConfig(config *tls.TLSConfig) *unwinder.TlsBinaryConfig {
	tlsConf := &unwinder.TlsBinaryConfig{}
	for idx, variable := range config.Variables {
		tlsConf.Offsets[idx] = variable.Offset
	}
	for idx := len(config.Variables); idx < len(tlsConf.Offsets); idx++ {
		tlsConf.Offsets[idx] = uint64(^int64(-1))
	}
	return tlsConf
}

func convertToUnwindPythonConfig(config *python.PythonConfig) *unwinder.PythonConfig {
	return python_agent.ParsePythonUnwinderConfig(config)
}

func convertToUnwindPthreadConfig(config *pthread.PthreadConfig) *unwinder.PthreadConfig {
	return &unwinder.PthreadConfig{
		KeyData: unwinder.PthreadKeyData{
			Size:        config.KeyData.Size,
			ValueOffset: config.KeyData.ValueOffset,
			SeqOffset:   config.KeyData.SeqOffset,
		},
		FirstSpecificBlockOffset:   config.FirstSpecificBlockOffset,
		SpecificArrayOffset:        config.SpecificArrayOffset,
		StructPthreadPointerOffset: config.StructPthreadPointerOffset,
		KeySecondLevelSize:         config.KeySecondLevelSize,
		KeyFirstLevelSize:          config.KeyFirstLevelSize,
		KeysMax:                    config.KeysMax,
	}
}

func convertToUnwindPhpConfig(config *php.PhpConfig) *unwinder.PhpConfig {
	return php_agent.ParsePhpUnwinderConfig(config)
}

func (m *BPFBinaryManager) MoveFromCache(a *Allocation) bool {
	return m.tables.MoveFromCache(a.UnwindTableAllocation)
}

func (m *BPFBinaryManager) MoveToCache(a *Allocation) bool {
	return m.tables.MoveToCache(a.UnwindTableAllocation)
}
