package slices_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/slices"
)

func TestShuffle(t *testing.T) {
	now := time.Now()

	origComplex := []time.Time{
		now,
		now.Add(1 * time.Second),
		now.Add(2 * time.Second),
		now.Add(3 * time.Second),
	}
	inputComplex := []time.Time{
		now,
		now.Add(1 * time.Second),
		now.Add(2 * time.Second),
		now.Add(3 * time.Second),
	}
	assert.NotEqual(t, origComplex, slices.Shuffle(inputComplex, nil))
}
