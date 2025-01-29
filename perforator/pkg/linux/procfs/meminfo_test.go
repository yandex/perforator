package procfs

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////////////

func TestMemInfo(t *testing.T) {
	for _, test := range []struct {
		name     string
		fs       fstest.MapFS
		error    string
		expected MemInfo
	}{
		{
			name: "ok",
			fs: fstest.MapFS{
				"meminfo": &fstest.MapFile{Mode: 0o666, Data: []byte(`MemTotal:       477792540 kB
MemFree:        56776312 kB
MemAvailable:   433669964 kB
MemKernel:      32972656 kB
Buffers:         8808356 kB
Cached:         350762920 kB
SwapCached:            0 kB
Active:         322087296 kB
Inactive:       61273532 kB
Active(anon):   28297116 kB
Inactive(anon):    33652 kB
Active(file):   293790180 kB
Inactive(file): 61239880 kB
Unevictable:     4682744 kB
Mlocked:         4682744 kB
SwapTotal:             0 kB
SwapFree:              0 kB
Dirty:              6792 kB
Writeback:             0 kB
AnonPages:      28472468 kB
Mapped:          8966608 kB
Shmem:             34444 kB
KReclaimable:   25661592 kB
Slab:           27056008 kB
SReclaimable:   25661592 kB
SUnreclaim:      1394416 kB
KernelStack:     1408832 kB
PageTables:      1792376 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:    238896268 kB
Committed_AS:   229834724 kB
VmallocTotal:   34359738367 kB
VmallocUsed:     1637068 kB
VmallocChunk:          0 kB
Percpu:          1976320 kB
HardwareCorrupted:     0 kB
AnonHugePages:   6498304 kB
ShmemHugePages:        0 kB
ShmemPmdMapped:        0 kB
FileHugePages:         0 kB
FilePmdMapped:         0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
Hugetlb:               0 kB
DirectMap4k:     5764944 kB
DirectMap2M:    295176192 kB
DirectMap1G:    186646528 kB`)},
			},
			expected: MemInfo{
				MemTotal: 477792540 * 1024,
			},
		},
		{
			name: "err",
			fs: fstest.MapFS{
				"meminfo": &fstest.MapFile{Mode: 0o666, Data: []byte(`MemTotal:       477792540 kB
MemFree:        56776312 xB
MemAvailable:   433669964 kB
MemKernel:      32972656 kB
Buffers:         8808356 kB
Cached:         350762920 kB
SwapCached:            0 kB
DirectMap1G:    186646528 kB`)},
			},
			error: "unsupported unit xB",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := procfs{test.fs}

			meminfo, err := p.GetMemInfo()

			if test.error != "" {
				require.ErrorContains(t, err, test.error)
				return
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, test.expected, *meminfo)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
