package profiler

import (
	"fmt"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/programstate"
)

type CgroupConfig struct {
	// Name of cgroup in freezer hierarchy
	// Example - `porto/ISS-AGENT--vla-web-search-tier0-at-wljxthrh-106/pod_agent_box_base/workload_hamster_start`
	Name string `yaml:"name"`

	// Labels to put into resulting profile
	Labels map[string]string `yaml:"labels,omitempty"`
}

type trackedCgroup struct {
	l               log.Logger
	conf            *CgroupConfig
	state           *programstate.State
	freezerCgroupID uint64
	sampleConsumer  SampleConsumer
}

func (p *Profiler) newTrackedCgroup(
	conf *CgroupConfig,
	sampleConsumer SampleConsumer,
	state *programstate.State,
	l log.Logger,
) (*trackedCgroup, error) {
	t := &trackedCgroup{
		l:              log.With(l, log.String("cgroup", conf.Name)),
		conf:           conf,
		state:          state,
		sampleConsumer: sampleConsumer,
	}

	return t, nil
}

func (t *trackedCgroup) Close() {
	err := t.state.RemoveTracedCgroup(t.freezerCgroupID)
	if err != nil {
		t.l.Warn("Failed to remove traced cgroup from the eBPF maps", log.Error(err))
	}
}

func (t *trackedCgroup) Open(name string, freezerCgroupID uint64) error {
	t.l.Info("Registered cgroup", log.UInt64("id", freezerCgroupID))

	err := t.state.AddTracedCgroup(freezerCgroupID)
	if err != nil {
		return fmt.Errorf("failed to add cgroup %q with id %d to eBPF maps: %w", name, freezerCgroupID, err)
	}

	t.freezerCgroupID = freezerCgroupID

	return nil
}
