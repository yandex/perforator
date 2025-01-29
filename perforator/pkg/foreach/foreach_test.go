package foreach

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilter(t *testing.T) {
	ints := make([]int, 0)
	for i := range 10 {
		ints = append(ints, i)
	}
	evenInts := Filter(ints, func(i int) bool {
		return i%2 == 0
	})
	require.Equal(t, []int{0, 2, 4, 6, 8}, evenInts)
}

func TestMap(t *testing.T) {
	ints := make([]int, 0)
	for i := range 5 {
		ints = append(ints, i)
	}
	mappedInts := Map(ints, func(i int) string {
		return fmt.Sprint(i)
	})
	require.Equal(t, mappedInts, []string{"0", "1", "2", "3", "4"}, mappedInts)
}
