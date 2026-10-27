package solomon

import (
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/nop"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/core/metrics/internal/pkg/metricsutil"
	"github.com/yandex/perforator/library/go/core/metrics/internal/pkg/registryutil"
)

var (
	_ metrics.Registry        = (*Registry)(nil)
	_ metrics.MetricsStreamer = (*Registry)(nil)
)

type Registry struct {
	separator          string
	prefix             string
	tags               map[string]string
	rated              bool
	trackRateStartTime bool
	useNameTag         bool
	startTime          uint32
	streamFormat       StreamFormat

	subregistries map[string]*Registry
	m             *sync.Mutex

	metrics *sync.Map

	logger log.Logger
}

func NewRegistry(opts *RegistryOpts) *Registry {
	r := &Registry{
		separator:    ".",
		useNameTag:   false,
		startTime:    uint32(time.Now().Unix()),
		streamFormat: StreamSpack,

		subregistries: make(map[string]*Registry),
		m:             new(sync.Mutex),

		metrics: new(sync.Map),
		logger:  new(nop.Logger),
	}

	if opts != nil {
		r.separator = string(opts.Separator)
		r.prefix = opts.Prefix
		r.tags = opts.Tags
		r.rated = opts.Rated
		r.trackRateStartTime = opts.TrackRateStartTime
		r.useNameTag = opts.UseNameTag
		r.streamFormat = opts.StreamFormat
		for _, collector := range opts.Collectors {
			collector(r)
		}
		r.logger = opts.Logger
	}

	return r
}

// Rated returns copy of registry with rated set to desired value.
func (r Registry) Rated(rated bool) metrics.Registry {
	return &Registry{
		separator:          r.separator,
		prefix:             r.prefix,
		tags:               r.tags,
		rated:              rated,
		trackRateStartTime: r.trackRateStartTime,
		useNameTag:         r.useNameTag,
		startTime:          r.startTime,

		subregistries: r.subregistries,
		m:             r.m,

		metrics: r.metrics,
		logger:  r.logger,
	}
}

// WithTags creates new sub-scope, where each metric has tags attached to it.
func (r Registry) WithTags(tags map[string]string) metrics.Registry {
	return r.newSubregistry(r.prefix, registryutil.MergeTags(r.tags, tags))
}

// WithPrefix creates new sub-scope, where each metric has prefix added to it name.
func (r Registry) WithPrefix(prefix string) metrics.Registry {
	return r.newSubregistry(registryutil.BuildFQName(r.separator, r.prefix, prefix), r.tags)
}

// ComposeName builds FQ name with appropriate separator.
func (r Registry) ComposeName(parts ...string) string {
	return registryutil.BuildFQName(r.separator, parts...)
}

func (r Registry) AddMetric(metric Metric) Metric {
	return r.registerMetric(metric)
}

func (r Registry) Counter(name string) metrics.Counter {
	m := NewCounter(r.newMetricName(name), 0, WithTags(r.tags))
	m.useNameTag = r.useNameTag

	return r.registerMetric(&m).(metrics.Counter)
}

func (r Registry) FuncCounter(name string, function func() int64) metrics.FuncCounter {
	m := NewFuncCounter(r.newMetricName(name), function, WithTags(r.tags), WithNameTag(r.useNameTag))

	return r.registerMetric(&m).(metrics.FuncCounter)
}

func (r Registry) Gauge(name string) metrics.Gauge {
	m := NewGauge(r.newMetricName(name), 0, WithTags(r.tags), WithNameTag(r.useNameTag))

	return r.registerMetric(&m).(metrics.Gauge)
}

func (r Registry) FuncGauge(name string, function func() float64) metrics.FuncGauge {
	m := NewFuncGauge(r.newMetricName(name), function, WithTags(r.tags), WithNameTag(r.useNameTag))

	return r.registerMetric(&m).(metrics.FuncGauge)
}

func (r Registry) IntGauge(name string) metrics.IntGauge {
	m := NewIntGauge(r.newMetricName(name), 0, WithTags(r.tags), WithNameTag(r.useNameTag))

	return r.registerMetric(&m).(metrics.IntGauge)
}

func (r Registry) FuncIntGauge(name string, function func() int64) metrics.FuncIntGauge {
	m := NewFuncIntGauge(r.newMetricName(name), function, WithTags(r.tags), WithNameTag(r.useNameTag))

	return r.registerMetric(&m).(metrics.FuncIntGauge)
}

func (r Registry) Timer(name string) metrics.Timer {
	m := NewTimer(r.newMetricName(name), 0, WithTags(r.tags), WithNameTag(r.useNameTag))

	return r.registerMetric(&m).(metrics.Timer)
}

func (r Registry) Histogram(name string, buckets metrics.Buckets) metrics.Histogram {
	m := NewHistogram(
		r.newMetricName(name),
		metricsutil.BucketsBounds(buckets),
		make([]int64, buckets.Size()),
		0,
		WithTags(r.tags),
		WithNameTag(r.useNameTag),
	)

	return r.registerMetric(&m).(metrics.Histogram)
}

func (r Registry) DurationHistogram(name string, buckets metrics.DurationBuckets) metrics.Timer {
	m := NewHistogram(
		r.newMetricName(name),
		metricsutil.DurationBucketsBounds(buckets),
		make([]int64, buckets.Size()),
		0,
		WithTags(r.tags),
		WithNameTag(r.useNameTag),
	)

	return r.registerMetric(&m).(metrics.Timer)
}

func (r Registry) RemoveMetric(name string) {
	metricName := r.newMetricName(name)
	metricKey := r.metricKey(metricName)
	r.metrics.Delete(metricKey)
}

// RemoveMetricWithTags is used for deletion metrics added by Vectors
func (r Registry) RemoveMetricWithTags(name string, tags map[string]string) {
	metricName := r.newMetricName(name)

	registryTags := registryutil.MergeTags(r.tags, map[string]string{"rated": strconv.FormatBool(r.rated)})
	metricTags := registryutil.MergeTags(registryTags, tags)
	metricKey := registryutil.BuildRegistryKey(metricName, metricTags)

	r.metrics.Delete(metricKey)
}

func (r *Registry) newSubregistry(prefix string, tags map[string]string) *Registry {
	// differ simple and rated registries
	keyTags := registryutil.MergeTags(tags, map[string]string{"rated": strconv.FormatBool(r.rated)})
	registryKey := registryutil.BuildRegistryKey(prefix, keyTags)

	r.m.Lock()
	defer r.m.Unlock()

	if existing, ok := r.subregistries[registryKey]; ok {
		return existing
	}

	subregistry := &Registry{
		separator:          r.separator,
		prefix:             prefix,
		tags:               tags,
		rated:              r.rated,
		trackRateStartTime: r.trackRateStartTime,
		useNameTag:         r.useNameTag,
		startTime:          r.startTime,
		streamFormat:       r.streamFormat,

		subregistries: r.subregistries,
		m:             r.m,

		metrics: r.metrics,
		logger:  r.logger,
	}

	r.subregistries[registryKey] = subregistry
	return subregistry
}

func (r *Registry) newMetricName(name string) string {
	return registryutil.BuildFQName(r.separator, r.prefix, name)
}

func (r *Registry) registerMetric(s Metric) Metric {
	if r.rated {
		Rated(s)
	}
	if r.trackRateStartTime {
		ensureRateStartTime(s)
	}

	key := r.metricKey(s.getID())

	oldMetric, loaded := r.metrics.LoadOrStore(key, s)
	if !loaded {
		return s
	}

	if reflect.TypeOf(oldMetric) == reflect.TypeOf(s) {
		if oldMetric.(Metric).isMemOnly() != s.isMemOnly() {
			r.logger.Error("cannot have the same metric with different memOnly flags", log.String("metric_id", s.getID()))
		}
		return oldMetric.(Metric)
	} else {
		r.metrics.Store(key, s)
		return s
	}
}

func ensureRateStartTime(s Metric) {
	switch m := s.(type) {
	case *Counter:
		m.ensureStartTime()
	case *FuncCounter:
		m.ensureStartTime()
	case *Histogram:
		m.ensureStartTime()
	}
}

func (r *Registry) unregisterMetric(s Metric) {
	if r.rated {
		Rated(s)
	}

	r.metrics.Delete(r.metricKey(s.getID()))
}

func (r *Registry) metricKey(metricID string) string {
	// differ simple and rated registries
	keyTags := registryutil.MergeTags(r.tags, map[string]string{"rated": strconv.FormatBool(r.rated)})
	return registryutil.BuildRegistryKey(metricID, keyTags)
}
