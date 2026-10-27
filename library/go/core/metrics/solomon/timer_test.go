package solomon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimer_RecordDuration(t *testing.T) {
	c := NewTimer("mytimer", 0, WithTags(map[string]string{"ololo": "trololo"}))

	c.RecordDuration(1 * time.Second)
	assert.Equal(t, 1*time.Second, c.value.Load())

	c.RecordDuration(42 * time.Millisecond)
	assert.Equal(t, 42*time.Millisecond, c.value.Load())
}

func TestTimerRated_MarshalJSON(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewTimer("mytimer", 42*time.Millisecond, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts), WithRated(true))

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"RATE","labels":{"ololo":"trololo","sensor":"mytimer"},"value":0.042,"ts":1577836800}`)
	assert.Equal(t, expected, b)
}

func TestNameTagTimer_MarshalJSON(t *testing.T) {
	c := NewTimer("mytimer", 42*time.Millisecond, WithTags(map[string]string{"ololo": "trololo"}), WithUseNameTag(), WithRated(true))

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"RATE","labels":{"name":"mytimer","ololo":"trololo"},"value":0.042}`)
	assert.Equal(t, expected, b)
}
