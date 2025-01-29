package labels

import (
	"strings"

	"github.com/google/pprof/profile"
)

type ProcessInfo struct {
	Pid         *int64
	Tid         *int64
	Containers  []string
	ProcessName string
	ThreadName  string
}

func ExtractProcessInfo(sample *profile.Sample) *ProcessInfo {
	pi := &ProcessInfo{}

	if value, ok := sample.Label["workload"]; ok {
		// Skip first container for the nanny pods.
		// It contains pod name & configuration hash.
		// FIXME(sskvor): generalize container tree pruning (PERFORATOR-119).
		for _, container := range value {
			if strings.HasPrefix(container, "iss_hook_") {
				value = value[1:]
				break
			}
		}

		pi.Containers = append(pi.Containers, value...)
	}

	if pid, ok := sample.NumLabel["pid"]; ok {
		pi.Pid = &pid[0]
	}

	if tid, ok := sample.NumLabel["tid"]; ok {
		pi.Tid = &tid[0]
	}

	if comm, ok := sample.Label["process_comm"]; ok {
		pi.ProcessName = comm[0]
	}

	if comm, ok := sample.Label["thread_comm"]; ok {
		pi.ThreadName = sanitizeThreadName(comm[0])
	} else if comm, ok := sample.Label["comm"]; ok {
		pi.ThreadName = sanitizeThreadName(comm[0])
	}

	return pi
}

func sanitizeThreadName(name string) string {
	i := len(name)

	for ; i > 0; i-- {
		if name[i-1] < '0' || name[i-1] > '9' {
			break
		}
	}

	return name[:i]
}
