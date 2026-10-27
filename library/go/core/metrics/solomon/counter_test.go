package solomon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCounter_Add(t *testing.T) {
	c := NewCounter("mycounter", 0, WithTags(map[string]string{"ololo": "trololo"}))

	c.Add(1)
	assert.Equal(t, int64(1), c.value.Load())

	c.Add(42)
	assert.Equal(t, int64(43), c.value.Load())

	c.Add(1489)
	assert.Equal(t, int64(1532), c.value.Load())
}

func TestCounter_Inc(t *testing.T) {
	c := NewCounter("mycounter", 0, WithTags(map[string]string{"ololo": "trololo"}))

	for i := 0; i < 10; i++ {
		c.Inc()
	}
	assert.Equal(t, int64(10), c.value.Load())

	c.Inc()
	c.Inc()
	assert.Equal(t, int64(12), c.value.Load())
}

func TestCounter_getID(t *testing.T) {
	c := NewCounter("mycounter", 0, WithTags(map[string]string{"ololo": "trololo"}))

	assert.Equal(t, "mycounter", c.getID())
}

func TestCounter_getID_WithTS(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewCounter("mycounter", 0, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	assert.Equal(t, "mycounter(2020-01-01T00:00:00Z)", c.getID())
}

func TestCounter_MarshalJSON(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewCounter("mycounter", 42, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"COUNTER","labels":{"ololo":"trololo","sensor":"mycounter"},"value":42,"ts":1577836800}`)
	assert.Equal(t, expected, b)
}

func TestRatedCounter_MarshalJSON(t *testing.T) {
	c := NewCounter("mycounter", 42, WithTags(map[string]string{"ololo": "trololo"}), WithRated(true))

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"RATE","labels":{"ololo":"trololo","sensor":"mycounter"},"value":42}`)
	assert.Equal(t, expected, b)
}

func TestNameTagCounter_MarshalJSON(t *testing.T) {
	c := NewCounter("mycounter", 42, WithTags(map[string]string{"ololo": "trololo"}), WithUseNameTag())

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"COUNTER","labels":{"name":"mycounter","ololo":"trololo"},"value":42}`)
	assert.Equal(t, expected, b)
}
