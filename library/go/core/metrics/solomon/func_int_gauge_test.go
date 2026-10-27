package solomon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func TestFuncIntGauge_Value(t *testing.T) {
	val := new(atomic.Int64)
	c := NewFuncIntGauge("myintgauge", val.Load, WithTags(map[string]string{"ololo": "trololo"}))

	val.Store(1)
	assert.Equal(t, int64(1), c.Snapshot().(*IntGauge).value.Load())

	val.Store(42)
	assert.Equal(t, int64(42), c.Snapshot().(*IntGauge).value.Load())
}

func TestFuncIntGauge_getID(t *testing.T) {
	c := NewFuncIntGauge("myintgauge", func() int64 { return 0 }, WithTags(map[string]string{"ololo": "trololo"}))

	assert.Equal(t, "myintgauge", c.getID())
}

func TestFuncIntGauge_getID_WithTS(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewFuncIntGauge("myintgauge", func() int64 { return 0 }, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	assert.Equal(t, "myintgauge(2020-01-01T00:00:00Z)", c.getID())
}

func TestFuncIntGauge_MarshalJSON(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewFuncIntGauge("myintgauge", func() int64 { return 42 }, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"IGAUGE","labels":{"ololo":"trololo","sensor":"myintgauge"},"value":42,"ts":1577836800}`)
	assert.Equal(t, expected, b)
}

func TestNameTagFunIntGauge_MarshalJSON(t *testing.T) {
	c := NewFuncIntGauge("myintgauge", func() int64 { return 42 }, WithTags(map[string]string{"ololo": "trololo"}), WithUseNameTag())

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"IGAUGE","labels":{"name":"myintgauge","ololo":"trololo"},"value":42}`)
	assert.Equal(t, expected, b)
}
