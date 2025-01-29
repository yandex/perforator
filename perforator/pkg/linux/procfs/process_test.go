package procfs

import (
	_ "embed"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/linux"
)

////////////////////////////////////////////////////////////////////////////////

type mapfsBuilder struct {
	fs fstest.MapFS
}

func mfs() *mapfsBuilder {
	return &mapfsBuilder{make(fstest.MapFS)}
}

func (m *mapfsBuilder) add(path, body string) *mapfsBuilder {
	m.fs[path] = &fstest.MapFile{Mode: 0o444, Data: []byte(body)}
	return m
}

func (m *mapfsBuilder) done() fs.FS {
	return m.fs
}

////////////////////////////////////////////////////////////////////////////////

func TestProcessMappings(t *testing.T) {
	for _, test := range []struct {
		name     string
		fs       fstest.MapFS
		error    string
		expected []Mapping
	}{
		{
			name: "self",
			fs: fstest.MapFS{
				"self/maps": &fstest.MapFile{Mode: 0o666, Data: []byte(`563257d8f000-563257d91000 r--p 00000000 fd:01 1649                       /usr/bin/cat
563257d91000-563257d96000 r-xp 00002000 fd:01 1649                       /usr/bin/cat
563259694000-5632596b5000 rw-p 00000000 00:00 0                          [heap]
7f0aec0aa000-7f0aec0b0000 rw-p 00000000 00:00 0
7f0aec0be000-7f0aec0e1000 r-xp 00001000 fd:01 2825                       /usr/lib/x86_64-linux-gnu/ld-2.31.so
ffffffffff600000-ffffffffff601000 --xp 00000000 00:00 0                  [vsyscall]
`)},
			},
			expected: []Mapping{
				{
					Begin:       0x563257d8f000,
					End:         0x563257d91000,
					Permissions: MappingPermissionReadable | MappingPermissionPrivate,
					Device:      Device{253, 1},
					Inode:       Inode{1649, 0},
					Offset:      0,
					Path:        "/usr/bin/cat",
				},
				{
					Begin:       0x563257d91000,
					End:         0x563257d96000,
					Permissions: 21,
					Device:      Device{Maj: 253, Min: 1},
					Inode:       Inode{ID: 1649, Gen: 0},
					Offset:      8192,
					Path:        "/usr/bin/cat",
				},
				{
					Begin:       0x563259694000,
					End:         0x5632596b5000,
					Permissions: MappingPermissionRWP,
					Path:        "[heap]",
				},
				{
					Begin:       0x7f0aec0aa000,
					End:         0x7f0aec0b0000,
					Permissions: MappingPermissionRWP,
					Path:        "",
				},
				{
					Begin:       0x7f0aec0be000,
					End:         0x7f0aec0e1000,
					Permissions: MappingPermissionRXP,
					Device:      Device{Maj: 253, Min: 1},
					Inode:       Inode{ID: 2825, Gen: 0},
					Offset:      4096,
					Path:        "/usr/lib/x86_64-linux-gnu/ld-2.31.so",
				},
				{
					Begin:       0xffffffffff600000,
					End:         0xffffffffff601000,
					Permissions: MappingPermissionExecutable | MappingPermissionPrivate,
					Path:        "[vsyscall]",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := procfs{test.fs}

			mappings := make([]Mapping, 0)

			err := p.Self().ListMappings(func(m *Mapping) error {
				mappings = append(mappings, *m)
				return nil
			})

			if test.error != "" {
				require.ErrorContains(t, err, test.error)
				return
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, test.expected, mappings)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////

func TestEnvs(t *testing.T) {
	for _, test := range []struct {
		name     string
		fs       fstest.MapFS
		isError  bool
		expected map[string]string
	}{
		{
			name: "simple",
			fs: fstest.MapFS{
				"self/environ": &fstest.MapFile{Mode: 0o666, Data: []byte("FI=a\000SE=xyz")},
			},
			expected: map[string]string{
				"FI": "a",
				"SE": "xyz",
			},
		},
		{
			name:    "no_environ_file",
			fs:      fstest.MapFS{},
			isError: true,
		},
		{
			name: "empty",
			fs: fstest.MapFS{
				"self/environ": &fstest.MapFile{Mode: 0o666, Data: []byte("")},
			},
			isError:  false,
			expected: map[string]string{},
		},
		{
			name: "incorrect_format_1",
			fs: fstest.MapFS{
				"self/environ": &fstest.MapFile{Mode: 0o666, Data: []byte("ABCD")},
			},
			isError: true,
		},
		{
			name: "empty_value",
			fs: fstest.MapFS{
				"self/environ": &fstest.MapFile{Mode: 0o666, Data: []byte("var=")},
			},
			isError: false,
			expected: map[string]string{
				"var": "",
			},
		},
		{
			name: "value_with_equal_sign",
			fs: fstest.MapFS{
				"self/environ": &fstest.MapFile{Mode: 0o666, Data: []byte("var=VALUE=x")},
			},
			isError: false,
			expected: map[string]string{
				"var": "VALUE=x",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := procfs{test.fs}

			envs, err := p.Self().ListEnvs()

			if test.isError {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, test.expected, envs)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////

//go:embed gotest/status1.txt
var status1 string

func TestGetNamespacedPID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		data     string
		expected linux.ProcessID
	}{
		{
			name:     "simple",
			data:     status1,
			expected: linux.ProcessID(1),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fs := &fstest.MapFS{
				"self/status": &fstest.MapFile{Mode: 0o444, Data: []byte(test.data)},
			}
			p := procfs{fs}
			pid, err := p.Self().GetNamespacedPID()
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, test.expected, pid)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
