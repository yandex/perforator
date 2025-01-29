package ccache

import (
	"testing"
	"time"

	"github.com/karlseguin/ccache/v3/assert"
)

func Test_Bucket_GetMissFromBucket(t *testing.T) {
	bucket := testBucket()
	assert.Nil(t, bucket.get("invalid"))
}

func Test_Bucket_GetHitFromBucket(t *testing.T) {
	bucket := testBucket()
	item := bucket.get("power")
	assertValue(t, item, "9000")
}

func Test_Bucket_DeleteItemFromBucket(t *testing.T) {
	bucket := testBucket()
	bucket.delete("power")
	assert.Nil(t, bucket.get("power"))
}

func Test_Bucket_SetsANewBucketItem(t *testing.T) {
	bucket := testBucket()
	item, existing := bucket.set("spice", "flow", time.Minute, false)
	assertValue(t, item, "flow")
	item = bucket.get("spice")
	assertValue(t, item, "flow")
	assert.Equal(t, existing, nil)
}

func Test_Bucket_SetsAnExistingItem(t *testing.T) {
	bucket := testBucket()
	item, existing := bucket.set("power", "9001", time.Minute, false)
	assertValue(t, item, "9001")
	item = bucket.get("power")
	assertValue(t, item, "9001")
	assertValue(t, existing, "9000")
}

func testBucket() *bucket[string] {
	b := &bucket[string]{lookup: make(map[string]*Item[string])}
	b.lookup["power"] = &Item[string]{
		key:   "power",
		value: "9000",
	}
	return b
}

func assertValue(t *testing.T, item *Item[string], expected string) {
	assert.Equal(t, item.value, expected)
}
