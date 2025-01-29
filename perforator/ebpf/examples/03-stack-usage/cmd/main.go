package main

import (
	"bytes"
	"os"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"

	"github.com/yandex/perforator/perforator/ebpf/examples/03-stack-usage/loader"
)

func main() {
	prog := loader.LoadProg(false)

	_ = os.WriteFile("foo.elf", prog, 0o644)

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
