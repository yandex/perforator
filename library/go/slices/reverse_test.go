package slices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/slices"
)

func TestReverse(t *testing.T) {
	val := []string{"1", "2", "4", "3"}
	expected := []string{"3", "4", "2", "1"}
	assert.Equal(t, expected, slices.Reverse(val))
}
