package xelf

import (
	"debug/elf"
	"errors"
	"fmt"
	"os"
)

func lookupSymbols(f *elf.File, symbolNames ...string) (map[string]*elf.Symbol, error) {
	symbolsToFind := make(map[string]struct{}, len(symbolNames))
	for _, name := range symbolNames {
		symbolsToFind[name] = struct{}{}
	}

	foundSymbols := make(map[string]*elf.Symbol)

	processSymbols := func(syms []elf.Symbol) {
		for i := range syms {
			s := &syms[i]
			if s.Size == 0 || s.Value == 0 {
				continue
			}

			if _, needed := symbolsToFind[s.Name]; needed {
				foundSymbols[s.Name] = s
				delete(symbolsToFind, s.Name)
				if len(symbolsToFind) == 0 {
					return
				}
			}
		}
	}

	dsyms, err := f.DynamicSymbols()
	if err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		return nil, fmt.Errorf("failed to get dynamic symbols: %w", err)
	}
	processSymbols(dsyms)
	if len(symbolsToFind) > 0 {
		syms, err := f.Symbols()
		if err != nil && !errors.Is(err, elf.ErrNoSymbols) {
			return nil, fmt.Errorf("failed to get symbols: %w", err)
		}
		processSymbols(syms)
	}

	return foundSymbols, nil
}

func findPhdrForVaddr(f *elf.File, vaddr uint64) *elf.ProgHeader {
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_LOAD || (prog.Flags&elf.PF_X) == 0 {
			continue
		}

		if prog.Vaddr <= vaddr && vaddr < (prog.Vaddr+prog.Memsz) {
			return &prog.ProgHeader
		}
	}

	return nil
}

// GetSymbolFileOffsets returns the file offsets for multiple symbols in an ELF file.
// It returns a map where keys are the symbol names and values are their file offsets.
// If a symbol is not found or its containing segment cannot be determined,
// it will be omitted from the result map. An error is returned if there are issues
// reading the ELF file itself or its symbol tables.
func GetSymbolFileOffsets(file *os.File, symbolNames ...string) (map[string]uint64, error) {
	f, err := elf.NewFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open ELF file: %w", err)
	}
	defer f.Close()

	foundSymbols, err := lookupSymbols(f, symbolNames...)
	if err != nil {
		return nil, err
	}

	if len(foundSymbols) == 0 {
		return map[string]uint64{}, nil
	}

	phdrs := parsePhdrs(f, loadablePhdrFilter)
	symbolOffsets := make(map[string]uint64)
	for name, symbol := range foundSymbols {
		symbolOffset, err := ELFVaddrToOffset(phdrs, symbol.Value)
		if err == nil {
			symbolOffsets[name] = symbolOffset
		}
	}

	return symbolOffsets, nil
}
