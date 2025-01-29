package cpulist

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	cpulistConfigured = "/sys/devices/system/cpu/possible"
	cpulistOnline     = "/sys/devices/system/cpu/online"
)

func ListConfiguredCPUs() ([]int, error) {
	return parseCPUList(cpulistConfigured)
}

func ListOnlineCPUs() ([]int, error) {
	return parseCPUList(cpulistOnline)
}

func parseCPUList(path string) ([]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseConfiguredCPUs(f)
}

func parseConfiguredCPUs(r io.Reader) ([]int, error) {
	res := []int{}
	var err error

	br := bufio.NewScanner(r)
	for br.Scan() {
		parts := strings.Split(br.Text(), ",")
		for _, part := range parts {
			if index := strings.IndexByte(part, '-'); index != -1 {
				first, err := strconv.Atoi(part[:index])
				if err != nil {
					return nil, err
				}
				last, err := strconv.Atoi(part[index+1:])
				if err != nil {
					return nil, err
				}
				for first <= last {
					res = append(res, first)
					first++
				}
			} else {
				cpu, err := strconv.Atoi(part)
				if err != nil {
					return nil, err
				}
				res = append(res, cpu)
			}
		}
	}

	if br.Err() != nil {
		return nil, err
	}

	return res, nil
}
