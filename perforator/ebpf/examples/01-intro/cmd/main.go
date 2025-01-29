package main

import (
	"bytes"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"

	"github.com/yandex/perforator/perforator/ebpf/examples/01-intro/loader"
)

func main() {
	prog := loader.LoadProg(false)

	spec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(prog))
	if err != nil {
		panic(err)
	}

	var objs loader.Objs
	err = spec.LoadAndAssign(&objs, nil)
	if err != nil {
		panic(err)
	}
	defer objs.Close()

	tp, err := link.Tracepoint("sched", "sched_switch", objs.Progs.TraceSchedSwitch, nil)
	if err != nil {
		panic(err)
	}
	defer tp.Close()

	for {
		time.Sleep(time.Hour)
	}
}
