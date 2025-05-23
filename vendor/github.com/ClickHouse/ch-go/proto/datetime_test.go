package proto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDateTime_ToDateTime(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		v := time.Unix(1546290000, 0).UTC()
		d := ToDateTime(v)
		assert.Equal(t, int32(1546290000), int32(d))
	})
	t.Run("Zero", func(t *testing.T) {
		v := time.Time{}
		d := ToDateTime(v)
		assert.Equal(t, int32(0), int32(d))
	})
}

func TestDateTime_Time(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		d := DateTime(1546290000)
		assert.Equal(t, d.Time().Unix(), int64(1546290000))
	})

	t.Run("Zero", func(t *testing.T) {
		d := DateTime(0)
		assert.Equal(t, d.Time().Unix(), int64(0))
	})

	t.Run("IsZero", func(t *testing.T) {
		d1 := DateTime(0)
		d2 := time.Time{}
		assert.Equal(t, d1.Time().IsZero(), false)
		assert.Equal(t, d2.IsZero(), true)
	})
}
