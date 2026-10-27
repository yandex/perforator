package solomon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func TestFuncGauge_Value(t *testing.T) {
	val := new(atomic.Float64)
	c := NewFuncGauge("mygauge", val.Load, WithTags(map[string]string{"ololo": "trololo"}))

	val.Store(1)
	assert.Equal(t, float64(1), c.Snapshot().(*Gauge).value.Load())

	val.Store(42)
	assert.Equal(t, float64(42), c.Snapshot().(*Gauge).value.Load())
}

func TestFuncGauge_getID(t *testing.T) {
	c := NewFuncGauge("mygauge", func() float64 { return 0 }, WithTags(map[string]string{"ololo": "trololo"}))

	assert.Equal(t, "mygauge", c.getID())
}

func TestFuncGauge_getID_WithTS(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewFuncGauge("mygauge", func() float64 { return 0 }, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	assert.Equal(t, "mygauge(2020-01-01T00:00:00Z)", c.getID())
}

func TestFunGauge_MarshalJSON(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewFuncGauge("mygauge", func() float64 { return 42.18 }, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"DGAUGE","labels":{"ololo":"trololo","sensor":"mygauge"},"value":42.18,"ts":1577836800}`)
	assert.Equal(t, expected, b)
}

func TestNameTagFunGauge_MarshalJSON(t *testing.T) {
	c := NewFuncGauge("mygauge", func() float64 { return 42.18 }, WithTags(map[string]string{"ololo": "trololo"}), WithUseNameTag())

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"DGAUGE","labels":{"name":"mygauge","ololo":"trololo"},"value":42.18}`)
	assert.Equal(t, expected, b)
}
