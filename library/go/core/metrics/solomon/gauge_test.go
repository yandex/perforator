package solomon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGauge_Add(t *testing.T) {
	c := NewGauge("mygauge", 0, WithTags(map[string]string{"ololo": "trololo"}))

	c.Add(1)
	assert.Equal(t, float64(1), c.value.Load())

	c.Add(42)
	assert.Equal(t, float64(43), c.value.Load())

	c.Add(14.89)
	assert.Equal(t, float64(57.89), c.value.Load())
}

func TestGauge_Set(t *testing.T) {
	c := NewGauge("mygauge", 0, WithTags(map[string]string{"ololo": "trololo"}))

	c.Set(1)
	assert.Equal(t, float64(1), c.value.Load())

	c.Set(42)
	assert.Equal(t, float64(42), c.value.Load())

	c.Set(14.89)
	assert.Equal(t, float64(14.89), c.value.Load())
}

func TestGauge_getID(t *testing.T) {
	c := NewGauge("mygauge", 0, WithTags(map[string]string{"ololo": "trololo"}))

	assert.Equal(t, "mygauge", c.getID())
}

func TestGauge_getID_WithTS(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewGauge("mygauge", 0, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	assert.Equal(t, "mygauge(2020-01-01T00:00:00Z)", c.getID())
}

func TestGauge_MarshalJSON(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewGauge("mygauge", 42.18, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"DGAUGE","labels":{"ololo":"trololo","sensor":"mygauge"},"value":42.18,"ts":1577836800}`)
	assert.Equal(t, expected, b)
}

func TestNameTagGauge_MarshalJSON(t *testing.T) {
	c := NewGauge("mygauge", 42.18, WithTags(map[string]string{"ololo": "trololo"}), WithUseNameTag())

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"DGAUGE","labels":{"name":"mygauge","ololo":"trololo"},"value":42.18}`)
	assert.Equal(t, expected, b)
}
