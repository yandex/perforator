package xelf

import (
	"debug/elf"
	"encoding/hex"
	"io"
	"os"
)

func GetBuildID(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	return ReadBuildID(f)
}

func ReadBuildID(r io.ReaderAt) (string, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	return parseBuildID(f)
}

func parseBuildID(f *elf.File) (string, error) {
	goid := ""
	gnuid := ""

	visitNote := func(rs io.ReadSeeker) {
		r := newElfNoteReader(f, rs)
		for r.Next() {
			note := r.Note()

			switch note.Name {
			case "Go":
				if note.Type == 4 && note.Description != nil {
					goid = string(note.Description)
				}
			case "GNU":
				if note.Type == NT_GNU_BUILD_ID && note.Description != nil {
					gnuid = parseGNUBuildID(note.Description)
				}
			}
		}
	}

	for _, scn := range f.Sections {
		if scn.Type != elf.SHT_NOTE {
			continue
		}

		visitNote(scn.Open())
	}

	if goid != "" {
		return goid, nil
	}
	if gnuid != "" {
		return gnuid, nil
	}

	for _, prog := range f.Progs {
		if prog.Type != elf.PT_NOTE {
			continue
		}
		visitNote(prog.Open())
	}

	if goid != "" {
		return goid, nil
	}
	if gnuid != "" {
		return gnuid, nil
	}

	// The binary probably doesn't have prepared build id.
	// Let's create some not-unique id by reading a few bits of .text.
	// It is not safe, there will be collisions, but for our simple use case
	// It should not matter.
	return PseudoBuildID(f)
}

func parseGNUBuildID(id []byte) string {
	return hex.EncodeToString(id)
}
