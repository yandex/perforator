package deploysystem

import (
	"context"
	"errors"
	"time"
)

// PollingSnapshotAdapter turns a one-shot snapshot source into a polling source.
// Like Source, it requires serial calls to Snapshot and owns no background tasks.
type PollingSnapshotAdapter struct {
	source         Source
	interval       time.Duration
	requestTimeout time.Duration
	nextPoll       time.Time
}

// NewPollingSnapshotAdapter polls immediately on the first call, then waits at
// least interval after each completed request, including failed requests.
// requestTimeout applies only to the wrapped source, which must honor its context.
func NewPollingSnapshotAdapter(source Source, interval, requestTimeout time.Duration) (*PollingSnapshotAdapter, error) {
	if source == nil {
		return nil, errors.New("polling snapshot source is nil")
	}
	if interval <= 0 {
		return nil, errors.New("snapshot polling interval must be positive")
	}
	if requestTimeout <= 0 {
		return nil, errors.New("snapshot request timeout must be positive")
	}
	return &PollingSnapshotAdapter{source: source, interval: interval, requestTimeout: requestTimeout}, nil
}

func (p *PollingSnapshotAdapter) Snapshot(ctx context.Context) (Snapshot, error) {
	if wait := time.Until(p.nextPoll); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return Snapshot{}, context.Cause(ctx)
		case <-timer.C:
		}
	}

	requestCtx, cancel := context.WithTimeout(ctx, p.requestTimeout)
	defer cancel()
	snapshot, err := p.source.Snapshot(requestCtx)
	p.nextPoll = time.Now().Add(p.interval)
	return snapshot, err
}
