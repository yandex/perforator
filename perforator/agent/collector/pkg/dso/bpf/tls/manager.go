package tls

import (
	"sync"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/tls"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

type BPFManager struct {
	l   log.Logger
	bpf *machine.BPF

	mutex sync.Mutex
}

func NewBPFManager(l log.Logger, bpf *machine.BPF) (*BPFManager, error) {
	return &BPFManager{
		l:   l,
		bpf: bpf,
	}, nil
}

func (m *BPFManager) Add(id uint64, info *tls.TLSConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	conf := &unwinder.TlsBinaryConfig{}

	for idx, variable := range info.Variables {
		conf.Offsets[idx] = variable.Offset
	}

	for idx := len(info.Variables); idx < len(conf.Offsets); idx++ {
		conf.Offsets[idx] = uint64(^int64(-1))
	}

	return m.bpf.AddTLSConfig(unwinder.BinaryId(id), conf)
}

func (m *BPFManager) Release(id uint64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	err := m.bpf.DeleteTLSConfig(unwinder.BinaryId(id))
	if err != nil {
		m.l.Error("Failed to delete tls config", log.Error(err))
	}
}
