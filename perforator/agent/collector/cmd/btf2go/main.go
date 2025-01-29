package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/btf"
)

func run() error {
	pkg := flag.String("package", "", "Generated package name")
	path := flag.String("elf", "", "Path to the compiled ebpf object file")
	prefix := flag.String("prefix", "", "Prefix to add to each user-defined type")
	output := flag.String("output", "", "Path to the generated file")
	flag.Parse()

	spec, err := ebpf.LoadCollectionSpec(*path)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	w, err := os.Create(*output)
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()

	wb := bufio.NewWriter(w)
	defer func() { _ = wb.Flush() }()

	f := NewFormatter(wb, spec.Types)

	f.SetPackage(*pkg)
	f.SetPrefix(*prefix)

	for _, m := range spec.Maps {
		f.AddPublicMap(m)
	}
	for _, p := range spec.Programs {
		f.AddProgram(p)
	}

	// Add exported types by BTF_EXPORT
	for iter := spec.Types.Iterate(); iter.Next(); {
		s, ok := iter.Type.(*btf.Struct)
		if !ok {
			continue
		}
		if !strings.HasPrefix(s.Name, "btf_export") {
			continue
		}

		for _, m := range s.Members {
			f.AddPublicType(m.Type)
		}
	}

	err = f.Print()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
