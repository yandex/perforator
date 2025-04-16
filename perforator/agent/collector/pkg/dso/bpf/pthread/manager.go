package pthread

import (
	"sync"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/pthread"
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

func (m *BPFManager) Add(id uint64, conf *pthread.PthreadConfig) error {
	if conf == nil {
		return nil
	}

	pthreadConfig := &unwinder.PthreadConfig{
		KeyData: unwinder.PthreadKeyData{
			Size:        conf.KeyData.Size,
			ValueOffset: conf.KeyData.ValueOffset,
			SeqOffset:   conf.KeyData.SeqOffset,
		},
		FirstSpecificBlockOffset:   conf.FirstSpecificBlockOffset,
		SpecificArrayOffset:        conf.SpecificArrayOffset,
		StructPthreadPointerOffset: conf.StructPthreadPointerOffset,
		KeySecondLevelSize:         conf.KeySecondLevelSize,
		KeyFirstLevelSize:          conf.KeyFirstLevelSize,
		KeysMax:                    conf.KeysMax,
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.bpf.AddPthreadConfig(unwinder.BinaryId(id), pthreadConfig)
}

func (m *BPFManager) Release(id uint64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	err := m.bpf.DeletePthreadConfig(unwinder.BinaryId(id))
	if err != nil {
		m.l.Error("Failed to delete pthread config", log.Error(err))
	}
}
