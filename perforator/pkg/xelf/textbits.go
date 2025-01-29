package xelf

import (
	"crypto/md5"
	"debug/elf"
	"encoding/hex"
	"math/rand"
)

// Change the seed with caution: pseudo build ids should not change.
const seed = 0xdead1ad4
const iters = 30
const bufsz = 4096

func PseudoBuildID(f *elf.File) (string, error) {
	rng := rand.New(rand.NewSource(seed))
	hash := md5.New()
	buf := make([]byte, bufsz)

	for _, scn := range f.Sections {
		if !isExecutableSection(scn) {
			continue
		}

		r := scn.ReaderAt
		if r == nil {
			// Compressed section?
			continue
		}

		// Some BOLTed binaries contain executable sections of zero size:
		//   [Nr] Name              Type            Address          Off    Size   ES Flg Lk Inf Al
		//   ...
		//   [27] .text.warm        PROGBITS        000000000060018e 00518e 000000 00  AX  0   0  1
		if scn.Size == 0 {
			hash.Write([]byte(scn.Name))
			continue
		}

		for i := 0; i < iters; i++ {
			off := rng.Uint64() % scn.Size
			n, _ := r.ReadAt(buf, int64(off))
			_, _ = hash.Write(buf[:n])
		}
	}

	return "pseudo" + hex.EncodeToString(hash.Sum(nil)), nil
}

func isExecutableSection(scn *elf.Section) bool {
	if scn.Type != elf.SHT_PROGBITS {
		return false
	}

	const expected = elf.SHF_ALLOC | elf.SHF_EXECINSTR
	return scn.Flags&expected == expected
}
