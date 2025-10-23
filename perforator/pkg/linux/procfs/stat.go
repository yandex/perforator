package procfs

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// Stat represents the data from /proc/stat
type Stat struct {
	// Boot time in seconds since Unix epoch
	Btime uint64
}

func (f *procfs) GetStat() (*Stat, error) {
	file, err := f.fs.Open("stat")
	if err != nil {
		return nil, fmt.Errorf("failed to open stat: %w", err)
	}
	defer file.Close()

	var stat Stat

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) == 0 {
			continue
		}

		if fields[0] == "btime" {
			if len(fields) < 2 {
				continue
			}
			stat.Btime, err = strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse btime: %w", err)
			}
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &stat, nil
}
