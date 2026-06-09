package cluster_top

import (
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
)

type workerMetrics struct {
	processingSuccess metrics.Timer
	processingFailed  metrics.Timer
	queueWait         metrics.Timer
	processedDone     metrics.Counter
	processedFailed   metrics.Counter
	processedEmpty    metrics.Counter
}

func newWorkerMetrics(reg xmetrics.Registry) *workerMetrics {
	r := reg.WithPrefix("cluster_top_worker")
	return &workerMetrics{
		processingSuccess: r.WithTags(map[string]string{"status": "success"}).Timer("jobs.processing.timer"),
		processingFailed:  r.WithTags(map[string]string{"status": "failed"}).Timer("jobs.processing.timer"),
		queueWait:         r.Timer("jobs.queue_wait.timer"),
		processedDone:     r.WithTags(map[string]string{"status": "done"}).Counter("jobs.processed.count"),
		processedFailed:   r.WithTags(map[string]string{"status": "failed"}).Counter("jobs.processed.count"),
		processedEmpty:    r.WithTags(map[string]string{"status": "empty"}).Counter("jobs.processed.count"),
	}
}

func (m *workerMetrics) recordJob(processingErr error, profilesProcessed int, stats *JobExecutionStats) {
	if stats == nil {
		return
	}

	if processingErr != nil {
		m.processingFailed.RecordDuration(stats.Duration)
		m.processedFailed.Inc()
		return
	}

	m.processingSuccess.RecordDuration(stats.Duration)
	if stats.QueueWait > 0 {
		m.queueWait.RecordDuration(stats.QueueWait)
	}

	if profilesProcessed == 0 {
		m.processedEmpty.Inc()
		return
	}

	m.processedDone.Inc()
}
