// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package norm

import (
	"testing"
	"unicode"
	"unicode/utf8"
)

// TestCase is used for most tests.
type TestCase struct {
	in  []rune
	out []rune
}

func runTests(t *testing.T, name string, fm Form, tests []TestCase) {
	rb := reorderBuffer{}
	rb.init(fm, nil)
	for i, test := range tests {
		rb.setFlusher(nil, appendFlush)
		for j, rune := range test.in {
			b := []byte(string(rune))
			src := inputBytes(b)
			info := rb.f.info(src, 0)
			if j == 0 {
				rb.ss.first(info)
			} else {
				rb.ss.next(info)
			}
			if rb.insertFlush(src, 0, info) < 0 {
				t.Errorf("%s:%d: insert failed for rune %d", name, i, j)
			}
		}
		rb.doFlush()
		was := string(rb.out)
		want := string(test.out)
		if len(was) != len(want) {
			t.Errorf("%s:%d: length = %d; want %d", name, i, len(was), len(want))
		}
		if was != want {
			k, pfx := pidx(was, want)
			t.Errorf("%s:%d: \nwas  %s%+q; \nwant %s%+q", name, i, pfx, was[k:], pfx, want[k:])
		}
	}
}

func TestFlush(t *testing.T) {
	const (
		hello = "Hello "
		world = "world!"
	)
	buf := make([]byte, maxByteBufferSize)
	p := copy(buf, hello)
	out := buf[p:]
	rb := reorderBuffer{}
	rb.initString(NFC, world)
	if i := rb.flushCopy(out); i != 0 {
		t.Errorf("wrote bytes on flush of empty buffer. (len(out) = %d)", i)
	}

	for i := range world {
		// No need to set streamSafe values for this test.
		rb.insertFlush(rb.src, i, rb.f.info(rb.src, i))
		n := rb.flushCopy(out)
		out = out[n:]
		p += n
	}

	was := buf[:p]
	want := hello + world
	if string(was) != want {
		t.Errorf(`output after flush was "%s"; want "%s"`, string(was), want)
	}
	if rb.nrune != 0 {
		t.Errorf("non-null size of info buffer (rb.nrune == %d)", rb.nrune)
	}
	if rb.nbyte != 0 {
		t.Errorf("non-null size of byte buffer (rb.nbyte == %d)", rb.nbyte)
	}
}

var insertTests = []TestCase{
	{[]rune{'a'}, []rune{'a'}},
	{[]rune{0x300}, []rune{0x300}},
	{[]rune{0x300, 0x316}, []rune{0x316, 0x300}}, // CCC(0x300)==230; CCC(0x316)==220
	{[]rune{0x316, 0x300}, []rune{0x316, 0x300}},
	{[]rune{0x41, 0x316, 0x300}, []rune{0x41, 0x316, 0x300}},
	{[]rune{0x41, 0x300, 0x316}, []rune{0x41, 0x316, 0x300}},
	{[]rune{0x300, 0x316, 0x41}, []rune{0x316, 0x300, 0x41}},
	{[]rune{0x41, 0x300, 0x40, 0x316}, []rune{0x41, 0x300, 0x40, 0x316}},
}

func TestInsert(t *testing.T) {
	runTests(t, "TestInsert", NFD, insertTests)
}

var decompositionNFDTest = []TestCase{
	{[]rune{0xC0}, []rune{0x41, 0x300}},
	{[]rune{0xAC00}, []rune{0x1100, 0x1161}},
	{[]rune{0x01C4}, []rune{0x01C4}},
	{[]rune{0x320E}, []rune{0x320E}},
	{[]rune("음ẻ과"), []rune{0x110B, 0x1173, 0x11B7, 0x65, 0x309, 0x1100, 0x116A}},
}

var decompositionNFKDTest = []TestCase{
	{[]rune{0xC0}, []rune{0x41, 0x300}},
	{[]rune{0xAC00}, []rune{0x1100, 0x1161}},
	{[]rune{0x01C4}, []rune{0x44, 0x5A, 0x030C}},
	{[]rune{0x320E}, []rune{0x28, 0x1100, 0x1161, 0x29}},
}

func TestDecomposition(t *testing.T) {
	runTests(t, "TestDecompositionNFD", NFD, decompositionNFDTest)
	runTests(t, "TestDecompositionNFKD", NFKD, decompositionNFKDTest)
}

var compositionTest = []TestCase{
	{[]rune{0x41, 0x300}, []rune{0xC0}},
	{[]rune{0x41, 0x316}, []rune{0x41, 0x316}},
	{[]rune{0x41, 0x300, 0x35D}, []rune{0xC0, 0x35D}},
	{[]rune{0x41, 0x316, 0x300}, []rune{0xC0, 0x316}},
	// blocking starter
	{[]rune{0x41, 0x316, 0x40, 0x300}, []rune{0x41, 0x316, 0x40, 0x300}},
	{[]rune{0x1100, 0x1161}, []rune{0xAC00}},
	// parenthesized Hangul, alternate between ASCII and Hangul.
	{[]rune{0x28, 0x1100, 0x1161, 0x29}, []rune{0x28, 0xAC00, 0x29}},
}

func TestComposition(t *testing.T) {
	runTests(t, "TestComposition", NFC, compositionTest)
}

// TestCompositionAfterHangul tests that a Hangul syllable at the start of a
// segment does not disable regular canonical composition for the rest of the
// segment. The reorderBuffer switches to Hangul mode as soon as it sees a Jamo,
// and used to drop every non-Hangul composition after it. See go.dev/issue/81021.
func TestCompositionAfterHangul(t *testing.T) {
	prefixes := []string{
		"\u1100\u1161",       // Jamo L V
		"\u1100\u1161\u11a8", // Jamo L V T
		"\uac00",             // Hangul syllable LV
		"\uac01",             // Hangul syllable LVT
		"\ud7a3",             // last Hangul syllable
	}
	forms := []struct {
		name string
		c, d Form
	}{
		{"NFC", NFC, NFD},
		{"NFKC", NFKC, NFKD},
	}
	for _, p := range prefixes {
		for _, f := range forms {
			for r := rune(0); r <= unicode.MaxRune; r++ {
				s := string(r)
				if !utf8.ValidString(s) {
					continue
				}
				want := f.c.String(s)
				in := f.d.String(s)
				if in == want {
					continue // nothing to recompose
				}
				// Skip characters that may legitimately merge with the
				// Hangul prefix rather than recompose among themselves.
				if c, _ := utf8.DecodeRuneInString(in); c >= jamoLBase && c <= 0x11ff ||
					c >= hangulBase && c < hangulEnd {
					continue
				}
				in, want = p+in, f.c.String(p)+want
				if got := f.c.String(in); got != want {
					t.Errorf("%s(%+q) = %+q; want %+q", f.name, in, got, want)
				}
			}
		}
	}
}

// TestCompositionBlockedByStarter tests that a starter blocks composition
// across it even when the starter itself takes no part in any composition and
// the runes between it and the following mark never combine backward.
// See go.dev/issue/81001.
func TestCompositionBlockedByStarter(t *testing.T) {
	tests := []struct {
		name    string
		f       Form
		in, out string
	}{
		// U+113C2 is a starter (ccc 0), so it blocks U+0300 from
		// composing with the "i" of the U+FB01 expansion.
		{"NFKC", NFKC, "\U00016d68\ufb01\U000113c2\u0300\u0316", "\U00016d68fi\U000113c2\u0316\u0300"},
		{"NFC", NFC, "i\U000113c2\u0300\u0316", "i\U000113c2\u0316\u0300"},
		// Without the intervening starter the composition must still happen.
		{"NFC", NFC, "i\u0300\u0316", "\u00ec\u0316"},
	}
	for _, test := range tests {
		if got := test.f.String(test.in); got != test.out {
			t.Errorf("%s.String(%+q) = %+q; want %+q", test.name, test.in, got, test.out)
		}
	}
}

// TestCompositionSupplementaryPlane tests that the recomposition map
// distinguishes runes outside the BMP. The map key used to truncate both
// runes to 16 bits, so supplementary-plane runes aliased BMP entries and
// composition invented characters unrelated to the input.
func TestCompositionSupplementaryPlane(t *testing.T) {
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if !utf8.ValidRune(r) {
			continue
		}
		for _, m := range []rune{0x0300, 0x0307, 0x093C, 0x11F41, 0x16D67} {
			in := string(r) + string(m)
			for _, f := range []struct {
				name string
				c, d Form
			}{{"NFC", NFC, NFD}, {"NFKC", NFKC, NFKD}} {
				// Normalization must preserve equivalence.
				if got := f.c.String(in); f.d.String(got) != f.d.String(in) {
					t.Errorf("%s(%+q) = %+q; not equivalent to input", f.name, in, got)
				}
			}
		}
	}
}
