package slices_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/slices"
)

func TestMap(t *testing.T) {
	t.Run("double", func(t *testing.T) {
		s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		res := slices.Map(s, func(v int) int {
			return v * 2
		})
		expected := []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}
		assert.Equal(t, expected, res)
	})
	t.Run("custom_type", func(t *testing.T) {
		type ohMyInts []int
		type ohMyFloats []float32

		s := ohMyInts{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		res := slices.Map(s, func(v int) float32 {
			return float32(v) / 2
		})
		expected := ohMyFloats{0.5, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5}
		assert.Equal(t, expected, ohMyFloats(res))
	})
}

func TestMutate(t *testing.T) {
	t.Run("double", func(t *testing.T) {
		s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		res := slices.Mutate(s, func(v int) int {
			return v * 2
		})
		expected := []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}
		assert.Equal(t, fmt.Sprintf("%p", s), fmt.Sprintf("%p", res), "slices have different pointers")
		assert.Equal(t, expected, res)
	})
}
