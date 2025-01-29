package main

import (
	"bytes"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"

	"github.com/yandex/perforator/perforator/ebpf/examples/02-array-of-maps/loader"
	"github.com/yandex/perforator/perforator/pkg/must"
)

func main() {
	must.Must(rlimit.RemoveMemlock())

	prog := loader.LoadProg(false)

	spec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(prog))
	must.Must(err)

	var objs loader.Objs
	must.Must(spec.LoadAndAssign(&objs, nil))
	defer objs.Close()

	im := spec.Maps["gigabytes"].InnerMap

	tp, err := link.Tracepoint("sched", "sched_switch", objs.Progs.TraceSchedSwitch, nil)
	must.Must(err)
	defer tp.Close()

	for i := uint32(0); true; i++ {
		time.Sleep(time.Second * 5)

		imc := im.Copy()
		m, err := ebpf.NewMap(imc)
		must.Must(err)

		fd := uint32(m.FD())
		must.Must(objs.Maps.Gigabytes.Put(&i, &fd))

		must.Must(m.Close())
	}
}
