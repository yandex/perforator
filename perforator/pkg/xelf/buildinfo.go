package xelf

import (
	"debug/elf"
	"io"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////

type BuildInfo struct {
	BuildID         string
	LoadBias        uint64
	FirstPhdrOffset uint64
	HasDebugInfo    bool
}

////////////////////////////////////////////////////////////////////////////////

func ReadBuildInfo(r io.ReaderAt) (*BuildInfo, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var bi BuildInfo

	bi.BuildID, err = parseBuildID(f)
	if err != nil {
		return nil, err
	}

	bi.LoadBias, err = parseLoadBias(f)
	if err != nil {
		return nil, err
	}

	bi.FirstPhdrOffset, err = parseFirstPhdrOffset(f)
	if err != nil {
		return nil, err
	}

	bi.HasDebugInfo, err = hasDebugInfo(f)
	if err != nil {
		return nil, err
	}

	return &bi, nil
}

////////////////////////////////////////////////////////////////////////////////

func parseLoadBias(f *elf.File) (uint64, error) {
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}

		if prog.Flags&elf.PF_X != elf.PF_X {
			continue
		}

		if prog.Align <= 1 {
			return prog.Vaddr, nil
		}

		// See https://refspecs.linuxbase.org/elf/gabi4+/ch5.pheader.html.
		// In position independent executables, p_vaddr does not have to be aligned.
		return prog.Vaddr & ^(prog.Align - 1), nil
	}

	return 0, nil
}

func parseFirstPhdrOffset(f *elf.File) (uint64, error) {
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}

		return prog.Vaddr, nil
	}

	return 0, nil
}

////////////////////////////////////////////////////////////////////////////////

func hasDebugInfo(f *elf.File) (bool, error) {
	for _, scn := range f.Sections {
		if strings.HasPrefix(scn.Name, ".debug") || strings.HasPrefix(scn.Name, ".zdebug") {
			return true, nil
		}
	}

	return false, nil
}

////////////////////////////////////////////////////////////////////////////////
