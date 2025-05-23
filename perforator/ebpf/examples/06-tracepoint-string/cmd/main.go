package main

import (
	"bytes"
	"errors"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"

	"github.com/yandex/perforator/perforator/ebpf/examples/06-tracepoint-string/loader"
)

func main() {
	_ = rlimit.RemoveMemlock()

	prog := loader.LoadProg(false)

	spec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(prog))
	if err != nil {
		panic(err)
	}

	var objs loader.Objs
	err = spec.LoadAndAssign(&objs, &ebpf.CollectionOptions{
		Programs: ebpf.ProgramOptions{
			LogLevel:     ebpf.LogLevelInstruction | ebpf.LogLevelStats,
			LogSizeStart: 1_000_000,
		},
	})
	if err != nil {
		var verr *ebpf.VerifierError
		if errors.As(err, &verr) {
			for _, line := range verr.Log {
				println(line)
			}
		}
		panic(err)
	}
	defer objs.Close()

	tp, err := link.AttachRawTracepoint(link.RawTracepointOptions{
		Name:    "cgroup_mkdir",
		Program: objs.TraceCgroupMkdir,
	})
	if err != nil {
		panic(err)
	}
	defer tp.Close()

	for {
		time.Sleep(time.Hour)
	}
}
