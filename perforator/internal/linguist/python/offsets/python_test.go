package offsets

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/python"
	"github.com/yandex/perforator/perforator/internal/linguist/common/offsetloader"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

// preparePythonOffsetsForTest sets up pythonOffsets with the given versions
// and sets a cleanup function that restores the original offsets after the test.
func preparePythonOffsetsForTest(t *testing.T, versions []*python.PythonVersion) {
	original := pythonOffsets
	t.Cleanup(func() {
		pythonOffsets = original
	})

	m := make(map[offsetloader.LanguageVersion]*unwinder.PythonInternalsOffsets)
	for _, v := range versions {
		key := offsetloader.NewLanguageVersion(v.Major, v.Minor, v.Micro)
		m[key] = &unwinder.PythonInternalsOffsets{}
	}
	pythonOffsets = offsetloader.NewFromMap(m)
}

func TestPythonInternalsOffsetsByVersion_ExactMatch(t *testing.T) {
	preparePythonOffsetsForTest(t, []*python.PythonVersion{
		{Major: 3, Minor: 12, Micro: 10},
	})

	version := &python.PythonVersion{Major: 3, Minor: 12, Micro: 10}

	offsets, err := PythonInternalsOffsetsByVersion(version)
	require.NoError(t, err)
	require.NotNil(t, offsets)
}

func TestPythonInternalsOffsetsByVersion_FallbackNewCPythonRelease(t *testing.T) {
	preparePythonOffsetsForTest(t, []*python.PythonVersion{
		{Major: 2, Minor: 7, Micro: 18},
		{Major: 3, Minor: 12, Micro: 11},
		{Major: 3, Minor: 12, Micro: 12},
	})

	version := &python.PythonVersion{Major: 3, Minor: 12, Micro: 13}

	offsets, err := PythonInternalsOffsetsByVersion(version)
	require.NoError(t, err)
	require.NotNil(t, offsets)

	nearestVersion := &python.PythonVersion{Major: 3, Minor: 12, Micro: 12}
	nearestOffsets, err := PythonInternalsOffsetsByVersion(nearestVersion)
	require.NoError(t, err)
	require.Equal(t, nearestOffsets, offsets)
}

func TestPythonInternalsOffsetsByVersion_NoFallbackForNewMinor(t *testing.T) {
	preparePythonOffsetsForTest(t, []*python.PythonVersion{
		{Major: 3, Minor: 12, Micro: 10},
	})

	version := &python.PythonVersion{Major: 3, Minor: 99, Micro: 0}

	offsets, err := PythonInternalsOffsetsByVersion(version)
	require.Error(t, err)
	require.Nil(t, offsets)
}

func TestIsVersionSupported_WithFallback(t *testing.T) {
	preparePythonOffsetsForTest(t, []*python.PythonVersion{
		{Major: 3, Minor: 12, Micro: 11},
		{Major: 3, Minor: 12, Micro: 12},
	})

	exactVersion := &python.PythonVersion{Major: 3, Minor: 12, Micro: 12}
	require.True(t, IsVersionSupported(exactVersion))

	oldVersion := &python.PythonVersion{Major: 3, Minor: 12, Micro: 10}
	require.True(t, IsVersionSupported(oldVersion))

	newMicroVersion := &python.PythonVersion{Major: 3, Minor: 12, Micro: 99}
	require.True(t, IsVersionSupported(newMicroVersion))

	newMinorVersion := &python.PythonVersion{Major: 3, Minor: 99, Micro: 0}
	require.False(t, IsVersionSupported(newMinorVersion))

	require.False(t, IsVersionSupported(nil))
}

func TestDecodeVersion(t *testing.T) {
	// Test encode/decode roundtrip
	testCases := []*python.PythonVersion{
		{Major: 3, Minor: 12, Micro: 12},
		{Major: 3, Minor: 11, Micro: 0},
		{Major: 2, Minor: 7, Micro: 18},
		{Major: 3, Minor: 0, Micro: 1},
		{Major: 3, Minor: 6, Micro: 15},
	}

	for _, version := range testCases {
		require.Equal(t, version, decodeVersion(offsetloader.NewLanguageVersion(version.Major, version.Minor, version.Micro)))
	}
}
