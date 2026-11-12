package profiler

import (
	"context"
	"fmt"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/pkg/linux/perfevent"
)

type perfEventProgramType int

const (
	genericPerfEventProgram perfEventProgramType = iota
	amdBRSProgram
)

type PerfEventID = perfevent.ID

// PerfEventManager is a wrapper around the perfevent.EventManager.
// Its main intent is to abstract the user of Profiler from BPF part (particularly - bpf program fd)
type PerfEventManager struct {
	bpf          *machine.BPF
	eventmanager *perfevent.Manager
}

func NewPerfEventManager(bpf *machine.BPF, em *perfevent.Manager) *PerfEventManager {
	return &PerfEventManager{
		bpf:          bpf,
		eventmanager: em,
	}
}

func (m *PerfEventManager) Open(target *perfevent.Target, options *perfevent.Options) (*PerfEvent, error) {
	return m.openImpl(target, options, genericPerfEventProgram)
}

func (m *PerfEventManager) openImpl(
	target *perfevent.Target,
	options *perfevent.Options,
	progType perfEventProgramType,
) (*PerfEvent, error) {
	bundle, err := m.eventmanager.Open(target, options, perfevent.OpenConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to open perf event bundle: %w", err)
	}

	return &PerfEvent{
		Bundle:   bundle,
		bpf:      m.bpf,
		progType: progType,
	}, nil
}

type PerfEvent struct {
	*perfevent.Bundle
	bpf      *machine.BPF
	progType perfEventProgramType
}

func (w *PerfEvent) Attach(ctx context.Context) error {
	var fd int
	switch w.progType {
	case amdBRSProgram:
		fd = w.bpf.AmdBRSProgramFD()
	case genericPerfEventProgram:
		fallthrough
	default:
		fd = w.bpf.ProfilerProgramFD()
	}

	return w.Bundle.AttachBPF(ctx, fd)
}
