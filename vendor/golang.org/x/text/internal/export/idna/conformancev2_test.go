// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package idna

import (
	"io"
	"strings"
	"testing"

	"golang.org/x/text/internal/gen"
	"golang.org/x/text/internal/testtext"
	"golang.org/x/text/internal/ucd"
)

func TestConformance(t *testing.T) {
	testtext.SkipIfNotLong(t)

	var r io.ReadCloser
	if UnicodeVersion == "15.0.0" {
		r = gen.OpenUnicodeFile("idna", "", "IdnaTestV2.txt")
	} else {
		r = gen.OpenUnicodeFile("", "", "idna/IdnaTestV2.txt")
	}
	defer r.Close()

	section := "main"
	p := ucd.New(r)
	// Strict profiles with all optional validation enabled.
	transitional := New(Transitional(true), VerifyDNSLength(true), BidiRule(), MapForLookup())
	nonTransitional := New(VerifyDNSLength(true), BidiRule(), MapForLookup())
	// Lax profiles with all optional validation disabled,
	// to verify we always perform required validation.
	transitionalLax := New(Transitional(true), MapForLookup(), VerifyDNSLength(false), CheckHyphens(false), CheckJoiners(false), StrictDomainName(false))
	nonTransitionalLax := New(MapForLookup(), VerifyDNSLength(false), CheckHyphens(false), CheckJoiners(false), StrictDomainName(false))
	for p.Next() {
		var (
			src          = def(unescape(p.String(0)), "")
			toUnicode    = def(unescape(p.String(1)), src)
			toUnicodeErr = p.String(2)
			toASCIIN     = def(unescape(p.String(3)), toUnicode)
			toASCIINErr  = def(p.String(4), toUnicodeErr)
			toASCIIT     = def(unescape(p.String(5)), toASCIIN)
			toASCIITErr  = def(p.String(6), toASCIINErr)
		)

		if UnicodeVersion == "15.0.0" {
			switch src {
			case "\u200c", "\u200d":
				continue // known failures
			}
		}

		doTest(t, nonTransitional.ToUnicode, section+":ToUnicode", src, toUnicode, toUnicodeErr)
		doTest(t, nonTransitional.ToASCII, section+":ToASCII:N", src, toASCIIN, toASCIINErr)
		doTest(t, transitional.ToASCII, section+":ToASCII:T", src, toASCIIT, toASCIITErr)

		if UnicodeVersion == "15.0.0" {
			continue
		}
		doTest(t, nonTransitionalLax.ToUnicode, section+":ToUnicode:lax", src, toUnicode, filterErr(toUnicodeErr))
		doTest(t, nonTransitionalLax.ToASCII, section+":ToASCII:N:lax", src, toASCIIN, filterErr(toASCIINErr))
		doTest(t, transitionalLax.ToASCII, section+":ToASCII:T:lax", src, toASCIIT, filterErr(toASCIITErr))
	}
}

func filterErr(errors string) string {
	errors = strings.Trim(errors, "[]")
	if errors == "" {
		return ""
	}
	var remaining []string
	for s := range strings.FieldsSeq(strings.ReplaceAll(errors, ",", " ")) {
		// As described in IdnaTestV2.txt, ignore these error codes when testing for
		// errors with optional validation disabled.
		switch {
		case s == "A4_1" || s == "A4_2" || s == "X4_2": // VerifyDnsLength
		case s == "V2" || s == "V3": // CheckHyphens
		case s[0] == 'C': // CheckJoiners
		case s[0] == 'B': // CheckBidi
		case s == "U1": // UseSTD3ASCIIRules
		default:
			remaining = append(remaining, s)
		}
	}
	if len(remaining) == 0 {
		return ""
	}
	return "[" + strings.Join(remaining, ", ") + "]"
}

func def(field, fallback string) string {
	if field == "" {
		return fallback
	}
	if field == `""` {
		return ""
	}
	return field
}
