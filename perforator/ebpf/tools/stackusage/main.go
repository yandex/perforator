package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"golang.org/x/exp/maps"
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
	for _, name := range keys {
		usage, err := stackusage(spec.Programs[name])
		if err != nil {
			return err
		}

		fmt.Printf("Program %s uses %d bytes of stack\n", name, usage)
	}

	return nil
}

func stackusage(prog *ebpf.ProgramSpec) (int, error) {
	calcer := stackusagecalcer{program: prog}
	return calcer.do()
}

type stackusagecalcer struct {
	program       *ebpf.ProgramSpec
	symbolOffsets map[string]int
}

type symbol struct {
	insn       int
	maxstack   int16
	references []string
}

func (c *stackusagecalcer) do() (usage int, err error) {
	c.symbolOffsets, err = c.program.Instructions.SymbolOffsets()
	if err != nil {
		return 0, err
	}

	// Locate eBPF program entrypoint name
	entrypoint := ""
	for symbol, offset := range c.symbolOffsets {
		if offset == 0 {
			entrypoint = symbol
		}
	}
	if entrypoint == "" {
		return 0, fmt.Errorf("failed to locate eBPF program entrypoint name")
	}

	symbols := make(map[string]*symbol)
	var sym *symbol
	for _, insn := range c.program.Instructions {
		if symname := insn.Symbol(); symname != "" {
			sym = &symbol{}
			symbols[symname] = sym
		}

		if sym == nil {
			panic("No symbol defined")
		}
		sym.insn++

		if insn.Src == asm.RFP || insn.Dst == asm.RFP {
			offset := insn.Offset
			if offset > 0 {
				return 0, fmt.Errorf("expected negative only offsets for r10 access")
			}
			sym.maxstack = max(sym.maxstack, -offset)
		}

		if insn.IsFunctionCall() {
			sym.references = append(sym.references, insn.Reference())
		}
	}

	return visit(entrypoint, symbols, 0, 0)
}

func visit(name string, symbols map[string]*symbol, depth, stack int) (int, error) {
	sym := symbols[name]
	if sym == nil {
		return 0, fmt.Errorf("unknown function call %s", name)
	}

	// See https://github.com/torvalds/linux/blob/e5b3efbe1ab1793bb49ae07d56d0973267e65112/kernel/bpf/verifier.c#L5863-L5872
	usage := roundup(max(1, int(sym.maxstack)), 32)
	stack += usage

	for range depth {
		fmt.Print("  ")
	}

	if stack >= 512 {
		fmt.Printf("fn <%s> with stack usage of %d bytes (%d bytes before rounding, \033[31;1m%d\033[0m bytes total, %d instructions)\n", name, usage, sym.maxstack, stack, sym.insn)
	} else {
		fmt.Printf("fn <%s> with stack usage of %d bytes (%d bytes before rounding, %d bytes total, %d instructions)\n", name, usage, sym.maxstack, stack, sym.insn)
	}

	maxusage := stack

	for _, callee := range sym.references {
		usage, err := visit(callee, symbols, depth+1, stack)
		if err != nil {
			return 0, err
		}
		maxusage = max(maxusage, usage)
	}

	return maxusage, nil
}

func roundup(a, b int) int {
	rem := a % b
	if rem == 0 {
		return a
	}
	return a + (b - rem)
}
