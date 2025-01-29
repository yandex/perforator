package perfmap

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type symbol struct {
	index  int
	name   string
	offset uint64
	size   uint64
}

// SegmentBegin implements disjointsegmentsets.Item
func (s symbol) SegmentBegin() uint64 {
	return s.offset
}

// SegmentEnd implements disjointsegmentsets.Item
func (s symbol) SegmentEnd() uint64 {
	return s.offset + s.size
}

// GenerationNumber implements disjointsegmentsets.Item
func (s symbol) GenerationNumber() int {
	return s.index
}

func parseHex(data string) (uint64, error) {
	data = strings.TrimPrefix(data, "0x")

	num, err := strconv.ParseUint(data, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse hex: %w", err)
	}
	return num, nil
}

func parse(reader io.Reader) ([]symbol, error) {
	syms := []symbol{}
	scanner := bufio.NewScanner(reader)
	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()
		items := strings.SplitN(line, " ", 3)
		if len(items) != 3 {
			return nil, fmt.Errorf("invalid line (does not contain 3 parts): %s", line)
		}
		offset, err := parseHex(items[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse offset: %w", err)
		}
		size, err := parseHex(items[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse size: %w", err)
		}
		name := items[2]
		s := symbol{
			index:  i,
			name:   name,
			offset: offset,
			size:   size,
		}
		syms = append(syms, s)
	}
	err := scanner.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to scan perf map: %w", err)
	}
	return syms, nil
}
