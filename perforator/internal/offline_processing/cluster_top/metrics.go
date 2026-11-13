package cluster_top

import (
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
)

type workerMetrics struct {
	buildInfoLookupErrors    metrics.Counter
	binariesWithBuildInfo    metrics.Counter
	binariesWithoutBuildInfo metrics.Counter
	readBuildInfo            metrics.Timer
	heartbeatErrors          metrics.Counter
	leaseLost                metrics.Counter
	finalizationErrors       metrics.Counter
	processingSuccess        metrics.Timer
	processingFailed         metrics.Timer
	queueWait                metrics.Timer
	processedDone            metrics.Counter
	processedFailed          metrics.Counter
	processedEmpty           metrics.Counter
	processedSkipped         metrics.Counter
}

func newWorkerMetrics(reg xmetrics.Registry) *workerMetrics {
	r := reg.WithPrefix("cluster_top_worker")
	return &workerMetrics{
		buildInfoLookupErrors:    r.Counter("build_info.lookup_errors.count"),
		binariesWithBuildInfo:    r.Counter("build_info.found.count"),
		binariesWithoutBuildInfo: r.Counter("build_info.unavailable.count"),
		readBuildInfo:            r.Timer("build_info.read.timer"),
		heartbeatErrors:          r.Counter("jobs.lease.heartbeat_errors.count"),
		leaseLost:                r.Counter("jobs.lease.lost.count"),
		finalizationErrors:       r.Counter("jobs.finalization_errors.count"),
		processingSuccess:        r.WithTags(map[string]string{"status": "success"}).Timer("jobs.processing.timer"),
		processingFailed:         r.WithTags(map[string]string{"status": "failed"}).Timer("jobs.processing.timer"),
		queueWait:                r.Timer("jobs.queue_wait.timer"),
		processedDone:            r.WithTags(map[string]string{"status": "done"}).Counter("jobs.processed.count"),
		processedFailed:          r.WithTags(map[string]string{"status": "failed"}).Counter("jobs.processed.count"),
		processedEmpty:           r.WithTags(map[string]string{"status": "empty"}).Counter("jobs.processed.count"),
		processedSkipped:         r.WithTags(map[string]string{"status": "skipped"}).Counter("jobs.processed.count"),
	}
}

func (m *workerMetrics) recordJob(status string, profilesProcessed int, stats *JobExecutionStats) {
	if stats == nil {
		return
	}

	m.buildInfoLookupErrors.Add(int64(stats.Metrics.BuildInfoLookupErrors))
	m.binariesWithBuildInfo.Add(int64(stats.Metrics.BinariesWithBuildInfo))
	m.binariesWithoutBuildInfo.Add(int64(stats.Metrics.BinariesWithoutBuildInfo))
	if stats.Stages.ReadBuildInfo > 0 {
		m.readBuildInfo.RecordDuration(stats.Stages.ReadBuildInfo)
	}

	if status == JobStatusFailed {
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

func (m *workerMetrics) recordSkipped(stats *JobExecutionStats) {
	if stats == nil {
		return
	}

	if stats.QueueWait > 0 {
		m.queueWait.RecordDuration(stats.QueueWait)
	}
	m.processedSkipped.Inc()
}
