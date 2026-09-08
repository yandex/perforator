package collector

import "github.com/yandex/perforator/library/go/core/metrics"

type collectorMetrics struct {
	deletedObjects metrics.Counter
	deleteErrors   metrics.Counter

	collectExpiredTimer metrics.Timer
	deleteTimer         metrics.Timer

	successIterationsTimer metrics.Timer
	failedIterationsTimer  metrics.Timer

	busy metrics.IntGauge
}

func newGcStorageMetrics(r metrics.Registry) *collectorMetrics {
	return &collectorMetrics{
		deletedObjects:         r.WithTags(map[string]string{"kind": "deleted"}).Counter("objects.count"),
		deleteErrors:           r.Counter("delete_error.count"),
		collectExpiredTimer:    r.Timer("collect_expired.timer"),
		deleteTimer:            r.Timer("delete.timer"),
		busy:                   r.IntGauge("busy.gauge"),
		successIterationsTimer: r.WithTags(map[string]string{"status": "success"}).Timer("iterations.timer"),
		failedIterationsTimer:  r.WithTags(map[string]string{"status": "failed"}).Timer("iterations.timer"),
	}
}
