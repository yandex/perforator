package cpulist

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListConfiguredCPUs(t *testing.T) {
	b := bytes.NewBufferString(`0-4,7-13,8,9,10`)
	cpus, err := parseConfiguredCPUs(b)
	require.NoError(t, err)
	require.Equal(t, cpus, []int{0, 1, 2, 3, 4, 7, 8, 9, 10, 11, 12, 13, 8, 9, 10})
}
