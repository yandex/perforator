package stackusage

import (
	"bytes"
	"fmt"
	"io"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
)

func StackUsage(prog *ebpf.ProgramSpec) (int, string, error) {
	log := bytes.NewBuffer(nil)
	calcer := stackusagecalcer{
		program: prog,
		log:     log,
	}

	usage, err := calcer.do()
	if err != nil {
		return 0, "", err
	}
	return usage, log.String(), nil
}

type stackusagecalcer struct {
	program       *ebpf.ProgramSpec
	symbolOffsets map[string]int
	log           io.Writer
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

	return visit(c.log, entrypoint, symbols, 0, 0)
}

func visit(log io.Writer, name string, symbols map[string]*symbol, depth, stack int) (int, error) {
	sym := symbols[name]
	if sym == nil {
		return 0, fmt.Errorf("unknown function call %s", name)
	}

	// See https://github.com/torvalds/linux/blob/e5b3efbe1ab1793bb49ae07d56d0973267e65112/kernel/bpf/verifier.c#L5863-L5872
	usage := roundup(max(1, int(sym.maxstack)), 32)
	stack += usage

	for range depth {
		_, err := fmt.Fprint(log, "  ")
		if err != nil {
			return 0, err
		}
	}

	var err error
	if stack >= 512 {
		_, err = fmt.Fprintf(log, "fn <%s> with stack usage of %d bytes (%d bytes before rounding, \033[31;1m%d\033[0m bytes total, %d instructions)\n", name, usage, sym.maxstack, stack, sym.insn)
	} else {
		_, err = fmt.Fprintf(log, "fn <%s> with stack usage of %d bytes (%d bytes before rounding, %d bytes total, %d instructions)\n", name, usage, sym.maxstack, stack, sym.insn)
	}
	if err != nil {
		return 0, err
	}

	maxusage := stack

	for _, callee := range sym.references {
		usage, err := visit(log, callee, symbols, depth+1, stack)
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
