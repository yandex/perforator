package python

import (
	"sync"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	pythonpreprocessing "github.com/yandex/perforator/perforator/agent/preprocessing/proto/python"
	python_agent "github.com/yandex/perforator/perforator/internal/linguist/python/agent"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

type BPFManager struct {
	l   log.Logger
	bpf *machine.BPF

	mutex sync.Mutex
}

func NewBPFManager(l log.Logger, bpf *machine.BPF) *BPFManager {
	return &BPFManager{
		l:   l,
		bpf: bpf,
	}
}

func (m *BPFManager) Add(id uint64, conf *pythonpreprocessing.PythonConfig) error {
	if conf == nil {
		return nil
	}

	pythonUnwindConfig := python_agent.ParsePythonUnwinderConfig(conf)
	if pythonUnwindConfig == nil {
		return nil
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.bpf.AddPythonConfig(unwinder.BinaryId(id), pythonUnwindConfig)
}

func (m *BPFManager) Release(id uint64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	err := m.bpf.DeletePythonConfig(unwinder.BinaryId(id))
	if err != nil {
		m.l.Error("Failed to delete python config", log.Error(err))
	}
}
