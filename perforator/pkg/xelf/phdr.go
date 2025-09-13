package xelf

import (
	"debug/elf"
	"errors"
)

type phdrFilter func(phdr *elf.ProgHeader) bool

func loadablePhdrFilter(phdr *elf.ProgHeader) bool {
	return phdr.Type == elf.PT_LOAD
}

func executablePhdrFilter(phdr *elf.ProgHeader) bool {
	return phdr.Type == elf.PT_LOAD && (phdr.Flags&elf.PF_X != 0)
}

// Phdr should match all filters to be included in the result
func parsePhdrs(f *elf.File, filters ...phdrFilter) (res []elf.ProgHeader) {
	for _, phdr := range f.Progs {
		matched := true
		for _, filter := range filters {
			if !filter(&phdr.ProgHeader) {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}

		res = append(res, phdr.ProgHeader)
	}

	return res
}

type alignedProgHeader struct {
	vaddr  uint64
	memsz  uint64
	offset uint64
	filesz uint64
}

func alignProgHeader(phdr *elf.ProgHeader) alignedProgHeader {
	return alignedProgHeader{
		vaddr:  phdr.Vaddr & ^(phdr.Align - 1),
		memsz:  phdr.Memsz + phdr.Vaddr%phdr.Align,
		offset: phdr.Off & ^(phdr.Align - 1),
		filesz: phdr.Filesz + phdr.Off%phdr.Align,
	}
}

func ELFOffsetToVaddr(phdrs []elf.ProgHeader, offset uint64) (vaddr uint64, err error) {
	for _, phdr := range phdrs {
		alignedPhdr := alignProgHeader(&phdr)

		if offset >= alignedPhdr.offset && offset-alignedPhdr.offset < alignedPhdr.filesz {
			vaddr = alignedPhdr.vaddr + offset - alignedPhdr.offset
			if vaddr-alignedPhdr.vaddr >= alignedPhdr.memsz {
				return 0, errors.New("computed vaddr is out of phdr.Memsz bounds")
			}

			return vaddr, nil
		}
	}

	return 0, errors.New("offset is not in any of the phdrs")
}

func ELFVaddrToOffset(phdrs []elf.ProgHeader, vaddr uint64) (offset uint64, err error) {
	for _, phdr := range phdrs {
		alignedPhdr := alignProgHeader(&phdr)

		if vaddr >= alignedPhdr.vaddr && vaddr-alignedPhdr.vaddr < alignedPhdr.memsz {
			offset = alignedPhdr.offset + vaddr - alignedPhdr.vaddr
			if offset-alignedPhdr.offset >= alignedPhdr.filesz {
				return 0, errors.New("computed offset is out of phdr.Filesz bounds")
			}

			return offset, nil
		}
	}

	return 0, errors.New("vaddr is not in any of the phdrs")
}
