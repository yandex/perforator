package jvmscanner

import "errors"

var errExcludedByte = errors.New("unexpected excluded byte in input")

type config struct {
	excluded uint8
}

var (
	u5config20p = config{excluded: 1} // JVM >= 20
	u5config19m = config{excluded: 0} // JVM <= 19
)

// u5decode reads one number from unsigned5-encoded buffer.
//
// On success return values are decoded value and the number of bytes comprising it.
func u5decode(cfg config, buf []byte) (uint32, int, error) {
	var acc uint32
	var i int
	var v byte
	var shift uint32 = 1
	for i, v = range buf {
		if v < cfg.excluded {
			return 0, 0, errExcludedByte
		}
		acc += (uint32(v) - uint32(cfg.excluded)) * shift
		shift *= 64
		if v <= 191 {
			break
		}
	}
	return acc, i + 1, nil
}
