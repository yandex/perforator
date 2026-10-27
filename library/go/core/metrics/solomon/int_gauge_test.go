package solomon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIntGauge_Add(t *testing.T) {
	c := NewIntGauge("myintgauge", 0, WithTags(map[string]string{"ololo": "trololo"}))

	c.Add(1)
	assert.Equal(t, int64(1), c.value.Load())

	c.Add(42)
	assert.Equal(t, int64(43), c.value.Load())

	c.Add(-45)
	assert.Equal(t, int64(-2), c.value.Load())
}

func TestIntGauge_Set(t *testing.T) {
	c := NewIntGauge("myintgauge", 0, WithTags(map[string]string{"ololo": "trololo"}))

	c.Set(1)
	assert.Equal(t, int64(1), c.value.Load())

	c.Set(42)
	assert.Equal(t, int64(42), c.value.Load())

	c.Set(-45)
	assert.Equal(t, int64(-45), c.value.Load())
}

func TestIntGauge_getID(t *testing.T) {
	c := NewIntGauge("myintgauge", 0, WithTags(map[string]string{"ololo": "trololo"}))

	assert.Equal(t, "myintgauge", c.getID())
}

func TestIntGauge_getID_WithTS(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewIntGauge("myintgauge", 0, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	assert.Equal(t, "myintgauge(2020-01-01T00:00:00Z)", c.getID())
}

func TestIntGauge_MarshalJSON(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewIntGauge("myintgauge", 42, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"IGAUGE","labels":{"ololo":"trololo","sensor":"myintgauge"},"value":42,"ts":1577836800}`)
	assert.Equal(t, expected, b)
}

func TestNameTagIntGauge_MarshalJSON(t *testing.T) {
	c := NewIntGauge("myintgauge", 42, WithTags(map[string]string{"ololo": "trololo"}), WithUseNameTag())

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"IGAUGE","labels":{"name":"myintgauge","ololo":"trololo"},"value":42}`)
	assert.Equal(t, expected, b)
}
