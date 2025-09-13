package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/cilium/ebpf"
	"golang.org/x/exp/maps"

	"github.com/yandex/perforator/perforator/pkg/ebpf/stackusage"
)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}

func run() error {
	path := flag.String("path", "", "Path to the eBPF ELF file")
	flag.Parse()

	f, err := os.Open(*path)
	if err != nil {
		return err
	}
	defer f.Close()

	spec, err := ebpf.LoadCollectionSpecFromReader(f)
	if err != nil {
		return err
	}

	keys := maps.Keys(spec.Programs)
	sort.Strings(keys)
	first := true
	for _, name := range keys {
		if first {
			first = false
		} else {
			fmt.Println()
		}
		usage, log, err := stackusage.StackUsage(spec.Programs[name])
		if err != nil {
			return err
		}
		fmt.Print(log)

		fmt.Printf("Program %s uses %d bytes of stack\n", name, usage)
	}

	return nil
}
