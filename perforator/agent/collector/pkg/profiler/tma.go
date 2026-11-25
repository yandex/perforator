package profiler

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/tma"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux/perfevent"
)

// parseScaledSlotCounts extracts the additional perf event deltas of the
// sample and scales them for event multiplexing. It returns nil when
// the data is partially or fully unavailable.
func parseScaledSlotCounts(sample *unwinder.RecordSampleParsed) []uint64 {
	deltas := sample.AdditionalEvents
	scaled := make([]uint64, len(deltas))
	for i, delta := range deltas {
		// TODO: this relies on a fact that additional_events has exactly as many elements as we have
		// TMA events.
		if delta == (unwinder.PerfEventValue{}) {
			return nil
		}
		scaled[i] = uint64(scalePerfCounterDelta(delta))
	}
	return scaled
}

// evaluateTmaMetrics computes the per-sample TMA metric values from the
// scaled additional-event values.
// Return value is nil when evaluation fails.
func evaluateTmaMetrics(hierarchy *tma.HierarchyInfo, scaled []uint64) []uint64 {
	if scaled[0] == 0 {
		return nil
	}

	metrics := hierarchy.Children
	values := make([]uint64, 0, len(metrics))
	for i := range metrics {
		// TODO: evaluation rules should be specified as part of tma.MetricDef instead.
		values = append(values, 100*scaled[i+1]/scaled[0])
	}
	return values
}

// processSample consumes the additional perf event deltas of the sample and
// returns the per-sample TMA metric values, or nil when the sample carries
// no TMA data.
func processSampleTMAEvents(hierarchy *tma.HierarchyInfo, sample *unwinder.RecordSampleParsed) []uint64 {
	scaled := parseScaledSlotCounts(sample)
	if scaled == nil {
		return nil
	}
	return evaluateTmaMetrics(hierarchy, scaled)
}

////////////////////////////////////////////////////////////////////////////////

// tmaEventManager owns the userspace side of TMA collection.
type tmaEventManager struct {
	events *perfevent.BundleGroup

	machineProfile *tma.MachineProfile
}

func (m *tmaEventManager) Close() error {
	return m.events.Close()
}

func validateTMAEvents(events []*tma.Event) error {
	const maxAdditionalEvents = len(unwinder.RecordSampleHeader{}.AdditionalEvents)
	if len(events) == 0 {
		return errors.New("machine profile has no perf events")
	}
	if len(events) > maxAdditionalEvents {
		return fmt.Errorf(
			"machine profile has too many perf events: %d, wire format allows %d",
			len(events),
			maxAdditionalEvents,
		)
	}
	return nil
}

func zipAdditionalEventFDs(perCPU []map[int]int) map[uint32][]uint32 {
	res := make(map[uint32][]uint32, len(perCPU[0]))
	for cpu, fd := range perCPU[0] {
		fds := make([]uint32, 0, len(perCPU))
		fds = append(fds, uint32(fd))
		for _, m := range perCPU[1:] {
			fds = append(fds, uint32(m[cpu]))
		}
		res[uint32(cpu)] = fds
	}
	return res
}

// setupTMA makes best effort to prepare for TMA metrics collection if the CPU and kernel
// support it.
func (p *Profiler) setupTMA(manager *perfevent.Manager) map[uint32][]uint32 {
	machineProfile, err := tma.Current()
	if err != nil {
		p.log.Warn("Failed to resolve TMA metric hierarchy, TMA analysis is disabled", log.Error(err))
		return nil
	}
	if machineProfile == nil {
		p.log.Info("Topdown metrics are not supported for current CPU, TMA analysis is disabled")
		return nil
	}

	if err := validateTMAEvents(machineProfile.Events); err != nil {
		p.log.Warn("Invalid TMA machine profile, TMA analysis is disabled", log.Error(err))
		return nil
	}

	var groupOpts []*perfevent.Options
	for _, ev := range machineProfile.Events {
		groupOpts = append(groupOpts, &perfevent.Options{
			Type: perfevent.Type{
				Type:   unix.PERF_TYPE_RAW,
				Config: ev.Config,
				Meta: perfevent.TypeMeta{
					Name: "tma-" + ev.Name,
					Unit: tma.SampleTypeUnit,
				},
			},
			Enable: true,
		})
	}

	openCfg := perfevent.OpenConfig{
		BestEffort: true,
	}

	group, err := manager.OpenGroup(
		&perfevent.Target{WholeSystem: true},
		groupOpts,
		openCfg,
	)
	if err != nil {
		p.log.Warn("Failed to open topdown events, TMA analysis is disabled", log.Error(err))
		return nil
	}

	members := group.Members()

	bundleFDs := make([]map[int]int, 0, len(members))
	for _, event := range members {
		bundleFDs = append(bundleFDs, event.FDs())
	}
	events := zipAdditionalEventFDs(bundleFDs)

	p.tma = &tmaEventManager{
		events:         group,
		machineProfile: machineProfile,
	}
	p.log.Info("Enabled TMA metrics collection", log.Int("cpus", len(events)))
	return events
}

////////////////////////////////////////////////////////////////////////////////
