package solomon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func TestFuncCounter_Inc(t *testing.T) {
	val := new(atomic.Int64)
	c := NewFuncCounter("mycounter", val.Load, WithTags(map[string]string{"ololo": "trololo"}))

	val.Store(1)
	assert.Equal(t, int64(1), c.Snapshot().(*Counter).value.Load())

	val.Store(42)
	assert.Equal(t, int64(42), c.Snapshot().(*Counter).value.Load())
}

func TestFuncCounter_getID(t *testing.T) {
	c := NewFuncCounter("mycounter", func() int64 { return 0 }, WithTags(map[string]string{"ololo": "trololo"}))

	assert.Equal(t, "mycounter", c.getID())
}

func TestFuncCounter_getID_WithTS(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewFuncCounter("mycounter", func() int64 { return 0 }, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	assert.Equal(t, "mycounter(2020-01-01T00:00:00Z)", c.getID())
}

func TestFuncCounter_MarshalJSON(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewFuncCounter("mycounter", func() int64 { return 42 }, WithTags(map[string]string{"ololo": "trololo"}), WithTimestamp(ts))

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"COUNTER","labels":{"ololo":"trololo","sensor":"mycounter"},"value":42,"ts":1577836800}`)
	assert.Equal(t, expected, b)
}

func TestRatedFuncCounter_MarshalJSON(t *testing.T) {
	c := NewFuncCounter("mycounter", func() int64 { return 42 }, WithTags(map[string]string{"ololo": "trololo"}), WithRated(true))

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"RATE","labels":{"ololo":"trololo","sensor":"mycounter"},"value":42}`)
	assert.Equal(t, expected, b)
}

func TestNameTagFuncCounter_MarshalJSON(t *testing.T) {
	c := NewFuncCounter("mycounter", func() int64 { return 42 }, WithTags(map[string]string{"ololo": "trololo"}), WithUseNameTag())

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := []byte(`{"type":"COUNTER","labels":{"name":"mycounter","ololo":"trololo"},"value":42}`)
	assert.Equal(t, expected, b)
}
