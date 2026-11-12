package perfevent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/linux/cpulist"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

////////////////////////////////////////////////////////////////////////////////

// The perf_event subsystem does not allow reinstalling eBPF programs after
// the first call to PERF_EVENT_IOC_SET_BPF ioctl, and there is no ioctl to
// detach a program — the kernel releases it only when the perf event is closed.
// So we respawn events in the userspace to replace a program (see reopen).
// Note that a respawned event gets a new fd and perf event id.

type factory struct {
	target  *Target
	options *Options
	leader  *Bundle
}

func (f *factory) open(cpu int) (*handle, error) {
	evTarget := &eventTarget{
		t:   *f.target,
		cpu: cpu,
	}
	evOptions := &eventOptions{
		o:             *f.options,
		groupLeaderFD: -1,
	}
	if f.leader != nil {
		evOptions.groupLeaderFD = f.leader.FDs()[cpu]
	}
	newEvent, err := newHandle(evTarget, evOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to open perf event at cpu %d: %w", cpu, err)
	}
	return newEvent, err
}

type cpuEvent struct {
	handle *handle
	cpu    int
}

// Bundle represents a mutable set of identically configured perf_event
// opened on different CPUs.
type Bundle struct {
	mgr     *Manager
	logger  xlog.Logger
	factory *factory
	// whether AttachBPF is allowed
	allowsAttach bool

	closed   atomic.Bool
	mu       sync.RWMutex
	events   []cpuEvent
	fds      map[int]int
	attached bool
}

// cpus returns the list of CPUs the bundle has events on.
func (b *Bundle) cpus() []int {
	cpus := make([]int, 0, len(b.events))
	for _, event := range b.events {
		cpus = append(cpus, event.cpu)
	}
	return cpus
}

// rebuildFDs updates fds field.
// It expectes that b.mu is read-locked.
func (b *Bundle) rebuildFDs() {
	res := make(map[int]int)
	for _, event := range b.events {
		res[event.cpu] = int(event.handle.fd.Fd())
	}
	b.fds = res
}

// FDs returns current file descriptors of all underlying events.
// Returned map may not be modified.
func (b *Bundle) FDs() map[int]int {
	res := make(map[int]int)
	b.mu.RLock()
	defer b.mu.RUnlock()
	return res
}

func foreach[E any](items []E, cb func(E) error) error {
	var errs []error
	for _, it := range items {
		errs = append(errs, cb(it))
	}
	return errors.Join(errs...)
}

func (b *Bundle) ContainsEvent(id ID) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, event := range b.events {
		if event.handle.id == id {
			return true
		}
	}
	return false
}

func (b *Bundle) Enable() error {
	return foreach(b.events, func(h cpuEvent) error {
		return h.handle.enable()
	})
}

func (b *Bundle) Disable() error {
	return foreach(b.events, func(h cpuEvent) error {
		return h.handle.disable()
	})
}

func (b *Bundle) reopen(ctx context.Context) (err error) {
	newEvents := make([]cpuEvent, 0, len(b.events))

	defer func() {
		if err == nil {
			return
		}
		closeErr := foreach(newEvents, func(h cpuEvent) error {
			return h.handle.close()
		})
		if closeErr != nil {
			b.logger.Warn(ctx, "Failed to close new events during failed reopen", log.Error(closeErr))
		}
	}()

	var oldIDs []ID
	for _, event := range b.events {
		newEvent, err := b.factory.open(event.cpu)
		if err != nil {
			return fmt.Errorf("failed to open new event: %w", err)
		}

		newEvents = append(newEvents, cpuEvent{
			handle: newEvent,
			cpu:    event.cpu,
		})
	}

	closeErr := foreach(b.events, func(h cpuEvent) error {
		id := h.handle.id
		e := h.handle.close()
		if e == nil {
			oldIDs = append(oldIDs, id)
		}
		return e
	})
	if closeErr != nil {
		b.logger.Warn(ctx, "Failed to close old events during reopen", log.Error(closeErr))
	}
	b.events = newEvents
	b.rebuildFDs()
	b.mgr.onReload(b, oldIDs)
	return nil
}

// attachBPF attaches the given eBPF program to every event of the bundle,
// replacing any program attached earlier.
func (b *Bundle) AttachBPF(ctx context.Context, progfd int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.attached {
		err := b.reopen(ctx)
		if err != nil {
			return fmt.Errorf("failed to recreate perf_events: %w", err)
		}
	}
	b.attached = true
	err := foreach(b.events, func(h cpuEvent) error {
		return h.handle.attachBPF(progfd)
	})
	if err != nil {
		return err
	}

	return nil
}

func (b *Bundle) closeImpl() error {
	return foreach(b.events, func(h cpuEvent) error {
		return h.handle.close()
	})
}

// Close destroys bundle.
// Any other call must complete before Close may be called.
// Close may not be called more than once.
func (b *Bundle) Close() error {
	if b.closed.Swap(true) {
		// double call to done will mess up tracker state,
		// let's prevent this
		panic("trying to close already closed bundle")
	}
	if !b.mu.TryLock() {
		panic("trying to close in-use bundle")
	}
	b.mgr.unregister(b)

	return b.closeImpl()
}

////////////////////////////////////////////////////////////////////////////////

// Manager manages all the perf events opened by the profiler.
type Manager struct {
	logger log.Logger
	cpus   []int

	eventCount       metrics.IntGauge
	eventCountByType metrics.IntGaugeVec
	mu               sync.RWMutex
	events           map[ID]*Bundle
}

func NewManager(l log.Logger, reg metrics.Registry) (*Manager, error) {
	l = l.WithName("perfevent")

	l.Info("Parsing CPU list")
	cpus, err := cpulist.ListOnlineCPUs()
	if err != nil {
		return nil, fmt.Errorf("failed to parse online CPUs: %w", err)
	}
	l.Info("Parsed CPU list", log.Int("cpu_count", len(cpus)))
	reg = reg.WithPrefix("perfevents")

	vec := reg.IntGaugeVec("event.count", []string{"type"})

	e := &Manager{
		logger:           l,
		cpus:             cpus,
		events:           make(map[ID]*Bundle),
		eventCountByType: vec,
		eventCount:       vec.With(map[string]string{"type": "-"}),
	}

	return e, nil
}

type openImplConfig struct {
	leader     *Bundle
	bestEffort bool
}

func (m *Manager) openImpl(
	target *Target,
	options *Options,
	cfg openImplConfig,
) (bundle *Bundle, err error) {
	var cpus []int
	if cfg.leader != nil {
		cpus = cfg.leader.cpus()
	} else {
		cpus = m.cpus
	}

	factory := &factory{
		options: options,
		target:  target,
		leader:  cfg.leader,
	}

	bundle = &Bundle{
		mgr:     m,
		logger:  xlog.Wrap(m.logger),
		factory: factory,
		events:  make([]cpuEvent, 0, len(cpus)),
	}
	if options.Frequency != nil || options.SampleRate != nil {
		bundle.allowsAttach = true
	}

	var errs []error

	for _, cpu := range cpus {
		var event *handle
		event, err = factory.open(cpu)

		if err != nil {
			if cfg.bestEffort {
				m.logger.Warn("Failed to open perf event, skipping CPU",
					log.Int("cpu", cpu),
					log.Any("type", options.Type),
					log.Error(err),
				)
				errs = append(errs, err)
				continue
			}

			closeErr := bundle.closeImpl()
			if closeErr != nil {
				m.logger.Warn("Failed to close event bundle", log.Error(closeErr))
			}
			return nil, err
		}

		bundle.events = append(bundle.events, cpuEvent{
			cpu:    cpu,
			handle: event,
		})
	}
	if len(bundle.events) == 0 {
		if len(errs) > 0 {
			return nil, fmt.Errorf("open failed on all CPUs: %w", errors.Join(errs...))
		}
		return nil, errors.New("no perf events were opened")
	}
	bundle.rebuildFDs()
	m.register(bundle)

	return bundle, nil
}

func (m *Manager) countEvents(count int64, typ Type) {
	m.eventCount.Add(count)
	m.eventCountByType.With(map[string]string{"type": typ.Name()}).Add(count)
}

func (m *Manager) register(b *Bundle) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, ev := range b.events {
		m.events[ev.handle.id] = b
	}
	m.countEvents(int64(len(b.events)), b.factory.options.Type)
}

func (m *Manager) unregister(b *Bundle) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.countEvents(-int64(len(b.events)), b.factory.options.Type)
	for _, ev := range b.events {
		delete(m.events, ev.handle.id)
	}
}

func (m *Manager) onReload(b *Bundle, oldEvents []ID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, ev := range oldEvents {
		delete(m.events, ev)
	}
	for _, ev := range b.events {
		m.events[ev.handle.id] = b
	}
}

type OpenConfig struct {
	BestEffort bool
}

func (m *Manager) Open(target *Target, options *Options, cfg OpenConfig) (*Bundle, error) {
	return m.openImpl(target, options, openImplConfig{
		bestEffort: cfg.BestEffort,
	})
}

type EventInfo struct {
	Type   Type
	Bundle *Bundle
}

func (m *Manager) GetEvent(id ID) *EventInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	bundle, ok := m.events[id]
	if !ok {
		return nil
	}

	return &EventInfo{
		Type:   bundle.factory.options.Type,
		Bundle: bundle,
	}
}
