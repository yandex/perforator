package profiler

import (
	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/cgroups"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
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
	bpf             *machine.BPF
	freezerCgroupID uint64
	builder         *multiProfileBuilder
}

func newTrackedCgroup(
	conf *CgroupConfig,
	bpf *machine.BPF,
	l log.Logger,
) (*trackedCgroup, error) {
	t := &trackedCgroup{
		l:       log.With(l, log.String("cgroup", conf.Name)),
		conf:    conf,
		bpf:     bpf,
		builder: newMultiProfileBuilder(conf.Labels),
	}

	return t, nil
}

func (t *trackedCgroup) Close() {
	err := t.bpf.RemoveTracedCgroup(t.freezerCgroupID)
	if err != nil {
		t.l.Warn("Failed to remove traced cgroup from the eBPF maps", log.Error(err))
	}
}

func (t *trackedCgroup) Open(name string, freezerCgroupID uint64) error {
	t.l.Info("Registered cgroup", log.UInt64("id", freezerCgroupID))

	err := t.bpf.AddTracedCgroup(freezerCgroupID)
	if err != nil {
		return err
	}

	t.freezerCgroupID = freezerCgroupID

	return nil
}

func (t *trackedCgroup) IsStale() bool {
	id, err := cgroups.GetCgroupID(t.conf.Name)
	if err != nil {
		return false
	}

	return t.freezerCgroupID != id
}
