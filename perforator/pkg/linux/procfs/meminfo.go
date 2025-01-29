package procfs

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

func GetMemInfo() (*MemInfo, error) {
	return FS().GetMemInfo()
}

func (f *procfs) GetMemInfo() (*MemInfo, error) {
	file, err := f.fs.Open("meminfo")
	if err != nil {
		return nil, fmt.Errorf("failed to open meminfo: %w", err)
	}
	defer file.Close()

	var meminfo MemInfo

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var (
			value uint64
			scale uint64 = 1
		)

		line := scanner.Text()
		fields := strings.Fields(line)

		switch len(fields) {
		case 2:
		case 3:
			switch fields[2] {
			case "B":
				scale = 1 << 0
			case "kB":
				scale = 1 << 10
			case "mB":
				scale = 1 << 20
			case "gB":
				scale = 1 << 30
			case "tB":
				scale = 1 << 40
			default:
				return nil, fmt.Errorf("malformed /proc/meminfo line %s: unsupported unit %s", line, fields[2])
			}
		default:
			return nil, fmt.Errorf("malformed /proc/meminfo line %s: unsupported number of fields", line)
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("malformed /proc/meminfo line %s: unsupported number of fields", line)
		}

		switch fields[0] {
		case "MemTotal:":
			meminfo.MemTotal = value * scale
		default:
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &meminfo, nil
}

type MemInfo struct {
	MemTotal uint64
}
