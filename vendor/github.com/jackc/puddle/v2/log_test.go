package puddle

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLog2Uint(t *testing.T) {
	r := require.New(t)

	r.Equal(uint8(0), log2Int(1))
	r.Equal(uint8(0), log2Int[uint64](1))
	r.Equal(uint8(1), log2Int[uint32](2))
	r.Equal(uint8(7), log2Int[uint8](math.MaxUint8))
	r.Equal(uint8(15), log2Int[uint16](math.MaxUint16))
	r.Equal(uint8(31), log2Int[uint32](math.MaxUint32))
	r.Equal(uint8(63), log2Int[uint64](math.MaxUint64))

	r.Panics(func() { log2Int[uint64](0) })
	r.Panics(func() { log2Int[int64](-1) })
}

func FuzzLog2Uint(f *testing.F) {
	const cnt = 1000

	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < cnt; i++ {
		val := uint64(rand.Int63())
		// val + 1 not to test val == 0.
		f.Add(val + 1)
	}

	f.Fuzz(func(t *testing.T, val uint64) {
		var mx uint8
		for i := 63; i >= 0; i-- {
			mask := uint64(1) << i
			if mask&val != 0 {
				mx = uint8(i)
				break
			}
		}

		require.Equal(t, mx, log2Int(val))
	})
}
