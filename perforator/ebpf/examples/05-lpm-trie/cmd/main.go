package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"

	"github.com/yandex/perforator/perforator/ebpf/examples/05-lpm-trie/loader"
)

func HostToBigEndian64(value uint64) uint64 {
	var buf [8]byte
	binary.NativeEndian.PutUint64(buf[:], value)
	return binary.BigEndian.Uint64(buf[:])
}

func main() {
	if err := rlimit.RemoveMemlock(); err != nil {
		panic(err)
	}

	prog := loader.LoadProg(true)

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

	for i := range 1024 * 1024 {
		prefix := uint64(i) << 44

		err = objs.Trie.Update(&loader.MappingTrieKey{
			Prefixlen:     32 + 20,
			Pid:           181223,
			AddressPrefix: HostToBigEndian64(prefix),
		}, &loader.MappingInfo{
			BinaryId: uint64(i),
		}, ebpf.UpdateAny)
		if err != nil {
			panic(err)
		}

		fmt.Printf("%x\n", prefix)
	}

	print("Start")

	for {
		time.Sleep(time.Hour)
	}
}
