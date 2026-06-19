package custom_profiling_operation

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindLibrariesByName(t *testing.T) {
	for _, test := range []struct {
		name        string
		fs          fstest.MapFS
		libraryName string
		expected    []string
	}{
		{
			name: "single_library",
			fs: fstest.MapFS{
				"proc/12345/maps": &fstest.MapFile{Mode: 0o444, Data: []byte(`7f0aec0be000-7f0aec0e1000 r-xp 00001000 fd:01 2825 /usr/lib/libcuda.so
`)},
			},
			libraryName: "libcuda.so",
			expected:    []string{"/usr/lib/libcuda.so"},
		},
		{
			name: "deduplicate_same_path",
			fs: fstest.MapFS{
				"proc/12345/maps": &fstest.MapFile{Mode: 0o444, Data: []byte(`7f0aec0be000-7f0aec0e1000 r-xp 00001000 fd:01 2825 /usr/lib/libcuda.so
7f0aec0e2000-7f0aec100000 r-xp 00002000 fd:01 2826 /usr/lib/libcuda.so
`)},
			},
			libraryName: "libcuda.so",
			expected:    []string{"/usr/lib/libcuda.so"},
		},
		{
			name: "versioned_library_name",
			fs: fstest.MapFS{
				"proc/12345/maps": &fstest.MapFile{Mode: 0o444, Data: []byte(`7f0aec0be000-7f0aec0e1000 r-xp 00001000 fd:01 2825 /opt/yt-nvidia/lib/driver/libcuda.so.580.65.06
`)},
			},
			libraryName: "libcuda.so",
			expected:    []string{"/opt/yt-nvidia/lib/driver/libcuda.so.580.65.06"},
		},
		{
			name: "versioned_cudart_library_name",
			fs: fstest.MapFS{
				"proc/12345/maps": &fstest.MapFile{Mode: 0o444, Data: []byte(`7f0aec0be000-7f0aec0e1000 r-xp 00001000 fd:01 2825 /usr/local/cuda-12.9/targets/x86_64-linux/lib/libcudart.so.12.9.37
`)},
			},
			libraryName: "libcudart.so",
			expected:    []string{"/usr/local/cuda-12.9/targets/x86_64-linux/lib/libcudart.so.12.9.37"},
		},
		{
			name: "not_executable_mapping_skipped",
			fs: fstest.MapFS{
				"proc/12345/maps": &fstest.MapFile{Mode: 0o444, Data: []byte(`7f0aec0be000-7f0aec0e1000 r--p 00001000 fd:01 2825 /usr/lib/libcuda.so
7f0aec0e2000-7f0aec100000 rw-p 00002000 fd:01 2826 /usr/lib/libcuda.so
7f0aec102000-7f0aec120000 r-xp 00003000 fd:01 2827 /usr/lib/libcuda.so
`)},
			},
			libraryName: "libcuda.so",
			expected:    []string{"/usr/lib/libcuda.so"},
		},
		{
			name: "not_found",
			fs: fstest.MapFS{
				"proc/12345/maps": &fstest.MapFile{Mode: 0o444, Data: []byte(`7f0aec0be000-7f0aec0e1000 r-xp 00001000 fd:01 2825 /usr/lib/libnvidia.so
`)},
			},
			libraryName: "libcuda.so",
			expected:    nil,
		},
		{
			name: "empty_maps",
			fs: fstest.MapFS{
				"proc/12345/maps": &fstest.MapFile{Mode: 0o444, Data: []byte(``)},
			},
			libraryName: "libcuda.so",
			expected:    nil,
		},
		{
			name: "no_executable_mappings",
			fs: fstest.MapFS{
				"proc/12345/maps": &fstest.MapFile{Mode: 0o444, Data: []byte(`7f0aec0be000-7f0aec0e1000 r--p 00001000 fd:01 2825 /usr/lib/libcuda.so
`)},
			},
			libraryName: "libcuda.so",
			expected:    nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testFS := &testProcFS{fs: test.fs}
			paths, err := findLibrariesByNameFromFS(testFS, test.libraryName)

			require.NoError(t, err)
			assert.ElementsMatch(t, test.expected, paths)
		})
	}
}
