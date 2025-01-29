package filter

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	_ = Filter(&PullingFilter{})
)

type Microscope struct {
	Selector      *querylang.Selector
	TimestampFrom time.Time
	TimestampTo   time.Time
}

type Config struct {
	PullInterval           time.Duration `yaml:"pull_interval"`
	PullBatchSize          uint64        `yaml:"pull_batch_size"`
	ListMicroscopesRetries uint64        `yaml:"list_microscope_retries"`
	TotalMicroscopesLimit  uint64        `yaml:"microscopes_limit"`
}

func (c *Config) fillDefault() {
	if c.PullInterval == time.Duration(0) {
		c.PullInterval = time.Minute
	}
	if c.PullBatchSize == 0 {
		c.PullBatchSize = 1000
	}
	if c.TotalMicroscopesLimit == 0 {
		c.TotalMicroscopesLimit = 1000
	}
}

type pullerMetrics struct {
	totalMicroscopesLimit metrics.IntGauge
	collectedMicroscopes  metrics.IntGauge
	failedParseSelector   metrics.Counter
	failedUpdates         metrics.Counter
	successfulUpdates     metrics.Counter
	discoveredFilters     map[MicroscopeType]metrics.Counter
	updateIterationPeriod metrics.Timer
}

// Pulls microscopes in background goroutine with some interval.
// Provides currently active microscopes filter (maybe with some delay)
type PullingFilter struct {
	l       xlog.Logger
	storage microscope.Storage
	c       Config

	filter atomic.Pointer[Filter]

	metrics *pullerMetrics
}

func NewPullingFilter(
	l xlog.Logger,
	reg metrics.Registry,
	c Config,
	storage microscope.Storage,
) (*PullingFilter, error) {
	c.fillDefault()

	filter := &PullingFilter{
		l:       l.WithName("PullingFilter"),
		storage: storage,
		c:       c,
		filter:  atomic.Pointer[Filter]{},
		metrics: &pullerMetrics{
			totalMicroscopesLimit: reg.WithTags(map[string]string{
				"kind": "limit",
			}).IntGauge("microscopes.count"),
			collectedMicroscopes: reg.WithTags(map[string]string{
				"kind": "collected",
			}).IntGauge("microscopes.count"),
			failedParseSelector: reg.Counter("selector.failed_parse.count"),
			discoveredFilters: map[MicroscopeType]metrics.Counter{
				AbstractFilter: reg.WithTags(map[string]string{"kind": "discovered", "filter": "abstract"}).Counter("filters.count"),
				PodFilter:      reg.WithTags(map[string]string{"kind": "discovered", "filter": "pod"}).Counter("filters.count"),
				NodeFilter:     reg.WithTags(map[string]string{"kind": "discovered", "filter": "node"}).Counter("filters.count"),
				ServiceFilter:  reg.WithTags(map[string]string{"kind": "discovered", "filter": "service"}).Counter("filters.count"),
			},
			failedUpdates: reg.WithTags(map[string]string{
				"status": "failed",
			}).Counter("microscope.updates.count"),
			successfulUpdates: reg.WithTags(map[string]string{
				"status": "success",
			}).Counter("microscope.updates.count"),
			updateIterationPeriod: reg.Timer("microscope.updates.timer"),
		},
	}

	filter.metrics.totalMicroscopesLimit.Set(int64(c.TotalMicroscopesLimit))

	return filter, nil
}

func (p *PullingFilter) collectMicroscopes(ctx context.Context) ([]microscope.Microscope, error) {
	filters := &microscope.Filters{
		User:         microscope.AllUsers,
		StartsBefore: ptr.Time(time.Now().Add(p.c.PullInterval)),
		EndsAfter:    ptr.Time(time.Now()),
	}
	pagination := &util.Pagination{
		Limit: p.c.PullBatchSize,
	}
	if p.c.PullBatchSize > p.c.TotalMicroscopesLimit {
		pagination.Limit = p.c.TotalMicroscopesLimit
	}
	lastErrors := uint64(0)
	var lastErr error

	collectedMicroscopes := make([]microscope.Microscope, 0)
	for {
		scopes, err := p.storage.ListMicroscopes(ctx, filters, pagination)
		if err != nil {
			p.l.Error(ctx,
				"Failed to list microscopes",
				log.Error(err),
				log.UInt64("limit", pagination.Limit),
				log.UInt64("offset", pagination.Offset),
				log.Any("filters", filters),
			)
			lastErrors++
			lastErr = err

			if lastErrors > p.c.ListMicroscopesRetries {
				return nil, fmt.Errorf("failed list microscopes %d times, one of the errors: %w", lastErrors, lastErr)
			}

			continue
		} else {
			lastErrors = 0
			lastErr = nil
		}

		p.l.Debug(ctx, "Listed microscopes", log.Any("pagination", pagination), log.Int("scopes", len(scopes)))

		collectedMicroscopes = append(collectedMicroscopes, scopes...)
		if uint64(len(scopes)) < pagination.Limit {
			break
		}

		pagination.Offset += pagination.Limit
		if pagination.Limit+uint64(len(collectedMicroscopes)) > p.c.TotalMicroscopesLimit {
			pagination.Limit = p.c.TotalMicroscopesLimit - uint64(len(collectedMicroscopes))
		}

		if pagination.Limit == 0 {
			break
		}
	}

	p.l.Info(ctx, "Collected microscopes", log.Int("num", len(collectedMicroscopes)))

	return collectedMicroscopes, nil
}

func (p *PullingFilter) newFilter(ctx context.Context, collectedMicroscopes []microscope.Microscope) Filter {
	filters := map[MicroscopeType]*MapLabelFilter{
		PodFilter:     NewMapLabelFilter(PodFilter),
		NodeFilter:    NewMapLabelFilter(NodeFilter),
		ServiceFilter: NewMapLabelFilter(ServiceFilter),
	}

	for _, microscope := range collectedMicroscopes {
		l := p.l.With(log.String("selector", microscope.Selector))
		l.Debug(ctx, "Found microscope active selector")

		parsedSelector, err := profilequerylang.ParseSelector(microscope.Selector)
		if err != nil {
			p.l.Error(ctx,
				"Failed to parse selector from microscope storage",
				log.Error(err),
				log.String("selector", microscope.Selector),
			)
			p.metrics.failedParseSelector.Inc()
			continue
		}

		result := DeductMicroscope(parsedSelector)
		if counter, ok := p.metrics.discoveredFilters[result.Type]; ok {
			counter.Inc()
		}
		if result.Type == AbstractFilter {
			l.Debug(ctx, "Skipped abstract microscope filter")
			continue
		}

		l.Debug(ctx, "Added microscope filter")
		filters[result.Type].AddValue(result.Value, microscope.FromTS, microscope.ToTS)
	}

	filtersToCombine := make([]Filter, 0, len(filters))
	for _, filter := range filters {
		filter.Finalize()
		filtersToCombine = append(filtersToCombine, filter)
	}

	return NewCombinedFilter(filtersToCombine)
}

func (p *PullingFilter) runPullIteration(ctx context.Context) error {
	ts := time.Now()
	defer func() {
		p.metrics.updateIterationPeriod.RecordDuration(time.Since(ts))
	}()

	collectedMicroscopes, err := p.collectMicroscopes(ctx)
	if err != nil {
		return err
	}

	filter := p.newFilter(ctx, collectedMicroscopes)
	p.filter.Store(&filter)

	p.metrics.collectedMicroscopes.Set(int64(len(collectedMicroscopes)))

	return nil
}

func (p *PullingFilter) Run(ctx context.Context) {
	ticker := time.NewTicker(p.c.PullInterval)

	p.l.Info(ctx, "Start microscope pulling filter")

	for {
		err := p.runPullIteration(ctx)
		if err != nil {
			p.l.Error(ctx, "Failed to pull microscopes", log.Error(err))
			p.metrics.failedUpdates.Inc()
		} else {
			p.metrics.successfulUpdates.Inc()
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *PullingFilter) Filter(meta *meta.ProfileMetadata) bool {
	if filter := p.filter.Load(); filter != nil {
		return (*filter).Filter(meta)
	}

	return false
}
