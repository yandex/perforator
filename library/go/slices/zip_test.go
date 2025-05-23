package slices_test

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/slices"
)

func TestZip(t *testing.T) {
	assert.Equal(
		t,
		maps.Collect(slices.Zip([]int{1, 2}, []string{"a", "b"})),
		map[int]string{1: "a", 2: "b"},
	)
}

func TestSlicesToMap(t *testing.T) {
	assert.Equal(
		t,
		slices.ZipToMap([]int{1, 2}, []string{"a", "b"}),
		map[int]string{1: "a", 2: "b"},
	)
}
