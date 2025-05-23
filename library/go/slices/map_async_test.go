package slices_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/slices"
)

func TestMapAsync(t *testing.T) {
	s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	t.Run("double", func(t *testing.T) {
		res, err := slices.MapA(context.Background(), s, func(ctx context.Context, v int) (int, error) {
			return v * 2, nil
		})
		assert.NoError(t, err)
		expected := []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}
		assert.Equal(t, expected, res)
	})
	t.Run("with_limit", func(t *testing.T) {
		res, err := slices.MapA(context.Background(), s, func(ctx context.Context, v int) (int, error) {
			return v, nil
		}, slices.WithLimit(2))
		assert.NoError(t, err)
		assert.Equal(t, s, res)
	})
	t.Run("error", func(t *testing.T) {
		_, err := slices.MapA(context.Background(), s, func(ctx context.Context, v int) (int, error) {
			return v, errors.New("error")
		})
		assert.EqualError(t, err, "error")
	})
	t.Run("panic", func(t *testing.T) {
		_, err := slices.MapA(context.Background(), s, func(ctx context.Context, v int) (int, error) {
			panic("null pointer")
		})
		assert.Error(t, err)
	})
}
