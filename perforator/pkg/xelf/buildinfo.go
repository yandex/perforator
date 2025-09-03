package xelf

import (
	"debug/elf"
	"errors"
	"io"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////

type BuildInfo struct {
	BuildID                 string
	LoadBias                uint64
	FirstPhdr               *elf.ProgHeader
	ExecutableLoadablePhdrs []elf.ProgHeader
	HasDebugInfo            bool
}

////////////////////////////////////////////////////////////////////////////////

func ReadGnuDebugLink(r io.ReaderAt) (string, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	sec := f.Section(".gnu_debuglink")
	if sec == nil {
		return "", errors.New("no .gnu_debuglink section")
	}
	data, err := sec.Data()
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", errors.New("empty .gnu_debuglink section")
	}
	n := strings.IndexByte(string(data), 0)
	if n == -1 {
		return "", errors.New("invalid .gnu_debuglink: missing NUL terminator")
	}

	return string(data[:n]), nil
}

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

	bi.ExecutableLoadablePhdrs = parsePhdrs(f, executablePhdrFilter)
	for _, phdr := range bi.ExecutableLoadablePhdrs {
		// See https://refspecs.linuxbase.org/elf/gabi4+/ch5.pheader.html
		// "Otherwise, p_align should be a positive, integral power of 2, and p_vaddr should equal p_offset, modulo p_align"
		if phdr.Align > 1 && phdr.Vaddr%phdr.Align != phdr.Off%phdr.Align {
			return nil, errors.New("program header alignment invariant is violated")
		}
	}

	if len(bi.ExecutableLoadablePhdrs) > 0 {
		bi.LoadBias = calculateLoadBias(&bi.ExecutableLoadablePhdrs[0])
	}

	bi.FirstPhdr = parseFirstLoadablePhdrInfo(f)

	bi.HasDebugInfo, err = hasDebugInfo(f)
	if err != nil {
		return nil, err
	}

	return &bi, nil
}

////////////////////////////////////////////////////////////////////////////////

func calculateLoadBias(firstExecutableLoadablePhdr *elf.ProgHeader) uint64 {
	if firstExecutableLoadablePhdr == nil {
		return 0
	}

	return firstExecutableLoadablePhdr.Vaddr & ^(firstExecutableLoadablePhdr.Align - 1)
}

func parseFirstLoadablePhdrInfo(f *elf.File) *elf.ProgHeader {
	for _, p := range f.Progs {
		if !loadablePhdrFilter(&p.ProgHeader) {
			continue
		}

		return &p.ProgHeader
	}

	return nil
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
