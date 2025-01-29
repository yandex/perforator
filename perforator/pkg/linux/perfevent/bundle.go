package perfevent

import (
	"errors"
	"fmt"
	"sync"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/linux/cpulist"
)

////////////////////////////////////////////////////////////////////////////////

// The perf_event subsystem does not allow reinstalling eBPF programs
// after the first call to PERF_EVENT_IOC_SET_BPF ioctl.
// So we respawn events in the userspace.
type event struct {
	mu      sync.Mutex
	logger  log.Logger
	handle  *Handle
	target  *Target
	options *Options
	bpfprog *int
}

func (e *event) Reopen() error {
	l := log.With(e.logger,
		log.Any("target", e.target),
		log.Any("options", e.options),
	)

	l.Debug("Trying to open perf event")
	next, err := NewHandle(e.target, e.options)
	if err != nil {
		l.Error("Failed to open perf event", log.Error(err))
		return err
	}
	l.Debug("Successfully opened perf event", log.UInt64("id", next.ID()))

	defer func() {
		if err != nil {
			_ = next.Close()
		}
	}()

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.handle != nil {
		err = e.handle.Close()
		if err != nil {
			return err
		}
	}

	e.handle = next

	return nil
}

func (e *event) Close() error {
	return e.handle.Close()
}

func (e *event) Handle() *Handle {
	return e.handle
}

func (e *event) AttachBPF(progfd int) error {
	defer func() {
		e.bpfprog = &progfd
	}()

	if e.bpfprog == nil {
		return e.handle.AttachBPF(progfd)
	}

	err := e.Reopen()
	if err != nil {
		return err
	}

	err = e.handle.AttachBPF(progfd)
	if err != nil {
		return err
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

type EventBundle struct {
	typ    Type
	done   func()
	events []*event
}

func (e *EventBundle) foreach(cb func(e *event) error) error {
	errs := make([]error, 0, len(e.events))
	for _, event := range e.events {
		errs = append(errs, cb(event))
	}
	return errors.Join(errs...)
}

func (e *EventBundle) Enable() error {
	return e.foreach(func(e *event) error {
		return e.handle.Enable()
	})
}

func (e *EventBundle) Disable() error {
	return e.foreach(func(e *event) error {
		return e.handle.Disable()
	})
}

func (e *EventBundle) AttachBPF(progfd int) error {
	return e.foreach(func(e *event) error {
		return e.AttachBPF(progfd)
	})
}

func (e *EventBundle) Close() error {
	if e.done != nil {
		e.done()
	}
	return e.foreach(func(e *event) error {
		return e.Close()
	})
}

////////////////////////////////////////////////////////////////////////////////

// EventManager manages all the perf events opened by the profiler.
type EventManager struct {
	mu      sync.Mutex
	logger  log.Logger
	bundles map[*EventBundle]any
	cpus    []int

	eventCount       metrics.IntGauge
	eventCountByType metrics.IntGaugeVec
}

func NewEventManager(l log.Logger, r metrics.Registry) (*EventManager, error) {
	l = l.WithName("perfevent")
	r = r.WithPrefix("perfevent")

	l.Info("Parsing CPU list")
	cpus, err := cpulist.ListOnlineCPUs()
	if err != nil {
		return nil, fmt.Errorf("failed to parse online CPUs: %w", err)
	}
	l.Info("Parsed CPU list", log.Int("cpu_count", len(cpus)))

	e := &EventManager{
		logger:  l,
		bundles: make(map[*EventBundle]any),
		cpus:    cpus,
	}

	e.instrument(r.WithPrefix("perfevents"))

	return e, nil
}

func (e *EventManager) instrument(r metrics.Registry) {
	e.eventCountByType = r.IntGaugeVec("event.count", []string{"type"})
	e.eventCount = e.eventCountByType.With(map[string]string{"type": "-"})
}

func (e *EventManager) countEvents(count int64, typ Type) {
	e.eventCount.Add(count)
	e.eventCountByType.With(map[string]string{"type": string(typ)}).Add(count)
}

func (e *EventManager) Open(
	target *Target,
	options *Options,
) (bundle *EventBundle, err error) {
	var cpus []int
	if target.CPU != nil {
		cpus = []int{*target.CPU}
	} else {
		cpus = e.cpus
	}

	bundle = &EventBundle{
		typ:    options.Type,
		events: make([]*event, 0, len(cpus)),
	}

	for _, cpu := range cpus {
		cpu := cpu
		target := *target
		options := *options
		target.CPU = &cpu

		event := &event{logger: e.logger, target: &target, options: &options}
		err = event.Reopen()
		if err != nil {
			closeErr := bundle.Close()
			if closeErr != nil {
				e.logger.Warn("Failed to close event bundle", log.Error(closeErr))
			}
			return nil, err
		}

		bundle.events = append(bundle.events, event)
	}

	e.register(bundle)

	return bundle, nil
}

func (e *EventManager) register(b *EventBundle) {
	b.done = func() {
		e.unregister(b)
	}
	e.bundles[b] = nil
	e.countEvents(int64(len(b.events)), b.typ)
}

func (e *EventManager) unregister(b *EventBundle) {
	delete(e.bundles, b)
	e.countEvents(-int64(len(b.events)), b.typ)
}

////////////////////////////////////////////////////////////////////////////////
