package procfs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBootTimeInitialized(t *testing.T) {
	btime, err := GetBootTime()
	require.NoError(t, err)
	require.NotZero(t, btime)
}
