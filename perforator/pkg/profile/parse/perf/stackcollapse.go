package perf

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
)

// stackcollapse-perf.pl does not account event period
// Let's build something similar by hand
var eventRe = regexp.MustCompile(`^(?P<comm>\S.+?)\s+(?P<period>\d+)\s+(?P<event>\S+):\s*$`)

const (
	kernelStartAddress = 0xffffffff00000000
	kernelEndAddress   = 0xffffffffffe00000
	kernelModuleName   = "[kernel]"
)

func ParsePerfScript(r io.Reader) (*profile.Profile, error) {
	res := profile.NewBuilder()
	res.AddSampleType("cpu", "cycles")
	res.SetDefaultSampleType("cpu")

	var sample *profile.SampleBuilder
	flush := func() {
		if sample != nil {
			sample.Finish()
			sample = nil
		}
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<30)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			flush()
			continue
		}
		if sample == nil {
			// header
			match := eventRe.FindStringSubmatch(line)
			if match == nil {
				return nil, fmt.Errorf("perf: malformed perf script line %#v", line)
			}
			comm := match[1]
			period, err := strconv.ParseInt(match[2], 10, 64)
			if err != nil {
				return nil, err
			}
			sample = res.Add(0).AddValue(period).AddStringLabel("comm", comm)
		} else {
			// stack
			idx := strings.IndexByte(line, ' ')
			if idx == -1 {
				return nil, fmt.Errorf("perf: malformed perf script line %#v", line)
			}
			addr, err := strconv.ParseUint(line[:idx], 16, 64)
			if err != nil {
				return nil, fmt.Errorf("perf: cannot parse address %qv: %w", line[:idx], err)
			}
			symbol := line[idx+1:]

			loc := sample.AddNativeLocation(addr).AddFrame().SetName(symbol).Finish()
			if addr >= kernelStartAddress && addr < kernelEndAddress {
				loc.SetMapping().SetPath(kernelModuleName).Finish()
			}
			sample = loc.Finish()
		}
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	flush()

	return res.Finish(), nil
}
