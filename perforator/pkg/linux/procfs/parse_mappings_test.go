package procfs_test

import (
	"bufio"
	"bytes"
	"os"
	"testing"

	"github.com/yandex/perforator/library/go/test/yatest"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
)

func BenchmarkParseMappings(b *testing.B) {
	raw, err := os.ReadFile(yatest.WorkPath("maps/clear_maps"))
	if err != nil {
		panic(err)
	}

	path := "test_path/"
	for i := 0; i < b.N; i++ {
		scanner := bufio.NewScanner(bytes.NewBuffer(raw))
		for scanner.Scan() {
			var mapping procfs.Mapping
			err = procfs.ParseProcessMapping(&mapping, scanner.Bytes(), &path)
			if err != nil {
				panic(err)
			}
		}
	}
}
