package jvmscanner

import "errors"

var errNullByte = errors.New("unexpected zero byte in input")

// U5decode reads one number from unsigned5-encoded buffer.
//
// On success return values are decoded value and the number of bytes comprising it.
func u5decode(buf []byte) (uint32, int, error) {
	var acc uint32
	var i int
	var v byte
	var shift uint32 = 1
	for i, v = range buf {
		if v == 0 {
			return 0, 0, errNullByte
		}
		acc += (uint32(v) - 1) * shift
		shift *= 64
		if v <= 191 {
			break
		}
	}
	return acc, i + 1, nil
}
