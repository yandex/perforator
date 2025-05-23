package proto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDateTime64_Time(t *testing.T) {
	for _, p := range []Precision{
		PrecisionSecond,
		1,
		PrecisionMilli,
		PrecisionMicro,
		PrecisionNano,
		8,
	} {
		t.Run(p.Duration().String(), func(t *testing.T) {
			for _, v := range []time.Time{
				time.Unix(0, 0).UTC(), // zero time
				time.Unix(1546290000, 0).UTC(),
			} {
				d := ToDateTime64(v, p)
				vt := d.Time(p)
				assert.Equal(t, ToDateTime64(v, p), d)
				assert.Equal(t, v.Unix(), vt.Unix())
				assert.True(t, p.Valid())
			}
		})

		t.Run("Zero_"+p.Duration().String(), func(t *testing.T) {
			t1 := time.Time{}
			t2 := time.Unix(0, 0).UTC()
			d1 := ToDateTime64(t1, p)
			d2 := ToDateTime64(t2, p)
			vt1 := d1.Time(p)
			vt2 := d2.Time(p)

			assert.True(t, t1.IsZero())
			assert.False(t, t2.IsZero())
			assert.Equal(t, d1, d2)
			assert.Equal(t, vt1.Unix(), int64(0))
			assert.Equal(t, vt2.Unix(), int64(0))
		})
	}
	t.Run("Duration", func(t *testing.T) {
		assert.Equal(t, time.Second, PrecisionSecond.Duration(), "sec")
		assert.Equal(t, time.Nanosecond, PrecisionNano.Duration(), "ns")
	})
}
