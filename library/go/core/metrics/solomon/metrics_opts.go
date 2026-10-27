package solomon

import "time"

type MetricsOpts struct {
	tags            map[string]string
	timestamp       *time.Time
	commonLabels    map[string]string
	startTime       uint32
	commonStartTime uint32
	useNameTag      bool
	memOnly         bool
	rated           bool
}

type MetricOpt func(*MetricsOpts)

func WithTags(tags map[string]string) func(*MetricsOpts) {
	return func(m *MetricsOpts) {
		m.tags = tags
	}
}

func WithUseNameTag() func(*MetricsOpts) {
	return func(m *MetricsOpts) {
		m.useNameTag = true
	}
}

func WithNameTag(useNameTag bool) func(*MetricsOpts) {
	return func(m *MetricsOpts) {
		m.useNameTag = useNameTag
	}
}

func WithTimestamp(t time.Time) func(*MetricsOpts) {
	return func(m *MetricsOpts) {
		m.timestamp = &t
	}
}

// WithStartTime sets the start time of a rate or rate histogram in Unix seconds.
func WithStartTime(seconds uint32) func(*MetricsOpts) {
	return func(m *MetricsOpts) {
		m.startTime = seconds
	}
}

// WithCommonStartTime sets the common start time used by SPACK 1.4.
func WithCommonStartTime(t time.Time) func(*MetricsOpts) {
	return func(m *MetricsOpts) {
		m.commonStartTime = uint32(t.Unix())
	}
}

func WithMemOnly() func(*MetricsOpts) {
	return func(m *MetricsOpts) {
		m.memOnly = true
	}
}

func WithRated(rated bool) func(*MetricsOpts) {
	return func(m *MetricsOpts) {
		m.rated = rated
	}
}

func WithCommonLabels(labels map[string]string) func(*MetricsOpts) {
	return func(m *MetricsOpts) {
		m.commonLabels = labels
	}
}
