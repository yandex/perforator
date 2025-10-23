package profiler

import (
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

////////////////////////////////////////////////////////////////////////////////

type trackedProcess struct {
	pid      linux.ProcessID
	features traceFeatures
	builder  *multiProfileBuilder
	bpf      *machine.BPF
}

func newTrackedProcess(
	pid linux.ProcessID,
	labels map[string]string,
	features traceFeatures,
	bpf *machine.BPF,
) (*trackedProcess, error) {
	err := bpf.AddTracedProcess(pid)
	if err != nil {
		return nil, err
	}

	return &trackedProcess{
		pid:      pid,
		features: features,
		builder:  newMultiProfileBuilder(labels),
		bpf:      bpf,
	}, nil
}

func (t *trackedProcess) traceFeatures() traceFeatures {
	return t.features
}

func (t *trackedProcess) close() error {
	return t.bpf.RemoveTracedProcess(t.pid)
}

////////////////////////////////////////////////////////////////////////////////
