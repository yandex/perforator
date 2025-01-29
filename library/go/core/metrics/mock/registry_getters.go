package mock

import (
	"github.com/yandex/perforator/library/go/core/metrics/internal/pkg/registryutil"
)

func (r *Registry) GetWithTags(tags map[string]string) (*Registry, bool) {
	return r.getSubRegistry(r.prefix, registryutil.MergeTags(r.tags, tags))
}

func (r *Registry) GetWithPrefix(prefix string) (*Registry, bool) {
	return r.getSubRegistry(registryutil.BuildFQName(r.separator, r.prefix, prefix), r.tags)
}

func (r *Registry) GetCounter(name string) (*Counter, bool) {
	return getMetric[Counter](r, name)
}

func (r *Registry) GetCounterVec(name string) (*CounterVec, bool) {
	return getMetric[CounterVec](r, name)
}

func (r *Registry) GetFuncCounter(name string) (*FuncCounter, bool) {
	return getMetric[FuncCounter](r, name)
}

func (r *Registry) GetGauge(name string) (*Gauge, bool) {
	return getMetric[Gauge](r, name)
}

func (r *Registry) GetGaugeVec(name string) (*GaugeVec, bool) {
	return getMetric[GaugeVec](r, name)
}

func (r *Registry) GetFuncGauge(name string) (*FuncGauge, bool) {
	return getMetric[FuncGauge](r, name)
}

func (r *Registry) GetIntGauge(name string) (*IntGauge, bool) {
	return getMetric[IntGauge](r, name)
}

func (r *Registry) GetIntGaugeVec(name string) (*IntGaugeVec, bool) {
	return getMetric[IntGaugeVec](r, name)
}

func (r *Registry) GetFuncIntGauge(name string) (*FuncIntGauge, bool) {
	return getMetric[FuncIntGauge](r, name)
}

func (r *Registry) GetTimer(name string) (*Timer, bool) {
	return getMetric[Timer](r, name)
}

func (r *Registry) GetTimerVec(name string) (*TimerVec, bool) {
	return getMetric[TimerVec](r, name)
}

func (r *Registry) GetHistogram(name string) (*Histogram, bool) {
	return getMetric[Histogram](r, name)
}

func (r *Registry) GetHistogramVec(name string) (*HistogramVec, bool) {
	return getMetric[HistogramVec](r, name)
}

func (r *Registry) GetDurationHistogram(name string) (*Histogram, bool) {
	return getMetric[Histogram](r, name)
}

func (r *Registry) GetDurationHistogramVec(name string) (*DurationHistogramVec, bool) {
	return getMetric[DurationHistogramVec](r, name)
}

func getMetric[T any](r *Registry, name string) (*T, bool) {
	key := registryutil.BuildRegistryKey(r.newMetricName(name), r.tags)
	val, ok := r.metrics.Load(key)
	if !ok {
		return nil, false
	}
	return val.(*T), ok
}

func (r *Registry) getSubRegistry(prefix string, tags map[string]string) (*Registry, bool) {
	registryKey := registryutil.BuildRegistryKey(prefix, tags)

	r.m.RLock()
	defer r.m.RUnlock()

	registry, ok := r.subregistries[registryKey]
	return registry, ok
}
