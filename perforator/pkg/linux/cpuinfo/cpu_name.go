package cpuinfo

import (
	"bufio"
	"bytes"
	"os"
	"regexp"
)

const (
	procCPUInfoPath = "/proc/cpuinfo"
)

var (
	// model name      : Intel(R) Xeon(R) Gold 6230 CPU @ 2.10GHz
	modelNameRgxp = regexp.MustCompile(`^model name\s*: (.*)$`)
)

func GetCPUModel() (string, error) {
	procbuf, err := os.ReadFile(procCPUInfoPath)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(bytes.NewBuffer(procbuf))
	for scanner.Scan() {
		matches := modelNameRgxp.FindStringSubmatch(scanner.Text())
		if matches == nil || len(matches) < 2 {
			continue
		}
		return matches[1], nil
	}
	return "Unknown CPU model", nil
}
