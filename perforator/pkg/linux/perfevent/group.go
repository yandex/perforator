package perfevent

import (
	"context"
	"fmt"

	"github.com/yandex/perforator/library/go/core/log"
)

// BundleGroup is group (in the perf_event sense) of several Bundles.
type BundleGroup struct {
	leader   *Bundle
	siblings []*Bundle
}

// OpenGroup opens a group of perf events on each requested CPU.
// Leader event is configured by the first element of `options` slice.
// OpenGroup expects that opts.leader is nil.
func (m *Manager) OpenGroup(
	target *Target,
	options []*Options,
	cfg OpenConfig,
) (_ *BundleGroup, err error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("invalid arguments: EventManager.OpenGroup requires at least one *Options")
	}
	var bundles []*Bundle
	defer func() {
		if err == nil {
			return
		}
		for _, b := range bundles {
			clerr := b.Close()
			if clerr != nil {
				m.logger.Warn("Failed to close event bundle after failed OpenGroup call", log.Error(clerr))
			}
		}
	}()

	leader, err := m.openImpl(target, options[0], openImplConfig{
		bestEffort: cfg.BestEffort,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open leader event bundle: %w", err)
	}
	bundles = append(bundles, leader)
	childOpts := openImplConfig{
		bestEffort: cfg.BestEffort,
		leader:     leader,
	}
	childOpts.leader = leader
	for _, opt := range options[1:] {
		var bundle *Bundle
		bundle, err = m.openImpl(target, opt, childOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to open sibling event bundle: %w", err)
		}
		bundles = append(bundles, bundle)
	}
	return &BundleGroup{
		leader:   leader,
		siblings: bundles[1:],
	}, nil
}

// AttachBPF attaches provided program to all events in the group, across all bundles.
// If it returns error, then some of the events may be in an inconsistent state.
func (bg *BundleGroup) AttachBPF(ctx context.Context, progfd int) error {
	err := bg.leader.AttachBPF(ctx, progfd)
	if err != nil {
		return err
	}
	for _, sibling := range bg.siblings {
		err := sibling.AttachBPF(ctx, progfd)
		if err != nil {
			return err
		}
	}
	return nil
}
