package custom_profiling_operation

import (
	"bufio"
	"fmt"
	"io/fs"
	"strings"

	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
)

type mappingIterator interface {
	ListMappings(callback func(m *procfs.Mapping) error) error
}

func findLibrariesByNameFromFS(fsf mappingIterator, name string) ([]string, error) {
	seen := make(map[string]struct{})

	err := fsf.ListMappings(func(m *procfs.Mapping) error {
		if m.Path == "" {
			return nil
		}

		if m.Permissions&procfs.MappingPermissionExecutable == 0 {
			return nil
		}

		if !strings.HasSuffix(m.Path, name) {
			return nil
		}

		if _, ok := seen[m.Path]; !ok {
			seen[m.Path] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list mappings: %w", err)
	}

	result := make([]string, 0, len(seen))
	for path := range seen {
		result = append(result, path)
	}
	return result, nil
}

func findLibrariesByName(pid linux.CurrentNamespacePID, name string) ([]string, error) {
	proc := procfs.Open(pid)
	return findLibrariesByNameFromFS(proc, name)
}

type testProcFS struct {
	fs fs.FS
}

func (t *testProcFS) ListMappings(callback func(m *procfs.Mapping) error) error {
	path := "proc/12345/maps"
	f, err := t.fs.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	s := bufio.NewScanner(bufio.NewReader(f))
	for s.Scan() {
		var mapping procfs.Mapping
		err = procfs.ParseProcessMapping(&mapping, s.Bytes(), &path)
		if err != nil {
			return err
		}

		err = callback(&mapping)
		if err != nil {
			return err
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	return nil
}
