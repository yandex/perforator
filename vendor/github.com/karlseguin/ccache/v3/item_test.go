package ccache

import (
	"math"
	"testing"
	"time"

	"github.com/karlseguin/ccache/v3/assert"
)

func Test_Item_Promotability(t *testing.T) {
	item := &Item[int]{promotions: 4}
	assert.Equal(t, item.shouldPromote(5), true)
	assert.Equal(t, item.shouldPromote(5), false)
}

func Test_Item_Expired(t *testing.T) {
	now := time.Now().UnixNano()
	item1 := &Item[int]{expires: now + (10 * int64(time.Millisecond))}
	item2 := &Item[int]{expires: now - (10 * int64(time.Millisecond))}
	assert.Equal(t, item1.Expired(), false)
	assert.Equal(t, item2.Expired(), true)
}

func Test_Item_TTL(t *testing.T) {
	now := time.Now().UnixNano()
	item1 := &Item[int]{expires: now + int64(time.Second)}
	item2 := &Item[int]{expires: now - int64(time.Second)}
	assert.Equal(t, int(math.Ceil(item1.TTL().Seconds())), 1)
	assert.Equal(t, int(math.Ceil(item2.TTL().Seconds())), -1)
}

func Test_Item_Expires(t *testing.T) {
	now := time.Now().UnixNano()
	item := &Item[int]{expires: now + (10)}
	assert.Equal(t, item.Expires().UnixNano(), now+10)
}

func Test_Item_Extend(t *testing.T) {
	item := &Item[int]{expires: time.Now().UnixNano() + 10}
	item.Extend(time.Minute * 2)
	assert.Equal(t, item.Expires().Unix(), time.Now().Unix()+120)
}
