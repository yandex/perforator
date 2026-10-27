package solomon

import (
	"encoding/json"
	"fmt"
	"maps"
	"time"
)

const (
	nameTag   = "name"
	sensorTag = "sensor"
)

type baseMetric struct {
	name       string
	metricType metricType
	tags       map[string]string
	timestamp  *time.Time
	startTime  uint32
	useNameTag bool
	memOnly    bool
}

func newBaseMetric(name string, mType metricType, opts ...MetricOpt) baseMetric {
	mOpts := MetricsOpts{}
	for _, op := range opts {
		op(&mOpts)
	}
	return baseMetric{
		name:       name,
		metricType: mType,
		tags:       mOpts.tags,
		timestamp:  mOpts.timestamp,
		startTime:  mOpts.startTime,
		useNameTag: mOpts.useNameTag,
		memOnly:    mOpts.memOnly,
	}
}

func (m *baseMetric) Name() string {
	return m.name
}

func (m *baseMetric) Labels() map[string]string {
	return m.tags
}

func (m *baseMetric) getType() metricType {
	return m.metricType
}

func (m *baseMetric) getTimestamp() *time.Time {
	return m.timestamp
}

func (m *baseMetric) getStartTime() uint32 {
	return m.startTime
}

func (m *baseMetric) ensureStartTime() {
	if m.startTime == 0 {
		m.startTime = uint32(time.Now().Unix())
	}
}

func (m *baseMetric) isMemOnly() bool {
	return m.memOnly
}

func (m *baseMetric) setMemOnly() {
	m.memOnly = true
}

func (m *baseMetric) getNameTag() string {
	if m.useNameTag {
		return nameTag
	}
	return sensorTag
}

func (m *baseMetric) getID() string {
	if m.timestamp != nil {
		return fmt.Sprintf("%s(%s)", m.name, m.timestamp.Format(time.RFC3339))
	}
	return m.name
}

func (m *baseMetric) labelsWithName() map[string]string {
	labels := make(map[string]string, len(m.tags)+1)
	labels[m.getNameTag()] = m.name
	maps.Copy(labels, m.tags)
	return labels
}

func (m *baseMetric) copy() baseMetric {
	return baseMetric{
		name:       m.name,
		metricType: m.metricType,
		tags:       m.tags,
		timestamp:  m.timestamp,
		startTime:  m.startTime,
		useNameTag: m.useNameTag,
		memOnly:    m.memOnly,
	}
}

func marshalScalarMetric(m *baseMetric, value any) ([]byte, error) {
	return json.Marshal(struct {
		Type      string            `json:"type"`
		Labels    map[string]string `json:"labels"`
		Value     any               `json:"value"`
		Timestamp *int64            `json:"ts,omitempty"`
		MemOnly   bool              `json:"memOnly,omitempty"`
	}{
		Type:      m.metricType.String(),
		Labels:    m.labelsWithName(),
		Value:     value,
		Timestamp: tsAsRef(m.timestamp),
		MemOnly:   m.memOnly,
	})
}
