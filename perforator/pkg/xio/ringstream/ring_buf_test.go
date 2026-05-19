package ringstream

import (
	"bytes"
	"testing"
)

func TestRingBuf_WriteReadNoWrap(t *testing.T) {
	r := newRingBuf(16)
	data := []byte("hello")
	r.write(0, data)

	got := make([]byte, len(data))
	r.read(0, got)
	if !bytes.Equal(got, data) {
		t.Fatalf("got %q, want %q", got, data)
	}
}

func TestRingBuf_WriteReadWrapAround(t *testing.T) {
	// capacity=8; write 5 bytes at offset 6 — wraps at byte 2.
	r := newRingBuf(8)
	data := []byte("ABCDE")
	r.write(6, data)

	got := make([]byte, len(data))
	r.read(6, got)
	if !bytes.Equal(got, data) {
		t.Fatalf("got %q, want %q", got, data)
	}
}

func TestRingBuf_MultipleRoundsWrap(t *testing.T) {
	const capacity = 8
	r := newRingBuf(capacity)

	// Write three rounds of 6 bytes each; each round crosses the wrap boundary.
	for round := 0; round < 3; round++ {
		off := int64(round * 6)
		src := make([]byte, 6)
		for i := range src {
			src[i] = byte('a' + round*6 + i)
		}
		r.write(off, src)

		dst := make([]byte, 6)
		r.read(off, dst)
		if !bytes.Equal(dst, src) {
			t.Fatalf("round %d: got %q, want %q", round, dst, src)
		}
	}
}

func TestRingBuf_SingleByteWrap(t *testing.T) {
	r := newRingBuf(4)
	// Write a single byte at the very last physical position.
	r.write(3, []byte{0xFF})
	got := make([]byte, 1)
	r.read(3, got)
	if got[0] != 0xFF {
		t.Fatalf("got %x, want ff", got[0])
	}
	// Write wrapping: offset 3 mod 4 = 3 (tail=1), so 2-byte write splits.
	r.write(3, []byte{0xAB, 0xCD})
	got2 := make([]byte, 2)
	r.read(3, got2)
	if !bytes.Equal(got2, []byte{0xAB, 0xCD}) {
		t.Fatalf("got %x, want abcd", got2)
	}
}
