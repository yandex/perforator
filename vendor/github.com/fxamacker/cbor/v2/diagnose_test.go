// Copyright (c) Faye Amacker. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package cbor

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestDiagnosticNotationExamples(t *testing.T) {
	// https://www.rfc-editor.org/rfc/rfc8949.html#name-examples-of-encoded-cbor-da
	testCases := []struct {
		data     []byte
		wantDiag string
	}{
		{
			data:     mustHexDecode("00"),
			wantDiag: `0`,
		},
		{
			data:     mustHexDecode("01"),
			wantDiag: `1`,
		},
		{
			data:     mustHexDecode("0a"),
			wantDiag: `10`,
		},
		{
			data:     mustHexDecode("17"),
			wantDiag: `23`,
		},
		{
			data:     mustHexDecode("1818"),
			wantDiag: `24`,
		},
		{
			data:     mustHexDecode("1819"),
			wantDiag: `25`,
		},
		{
			data:     mustHexDecode("1864"),
			wantDiag: `100`,
		},
		{
			data:     mustHexDecode("1903e8"),
			wantDiag: `1000`,
		},
		{
			data:     mustHexDecode("1a000f4240"),
			wantDiag: `1000000`,
		},
		{
			data:     mustHexDecode("1b000000e8d4a51000"),
			wantDiag: `1000000000000`,
		},
		{
			data:     mustHexDecode("1bffffffffffffffff"),
			wantDiag: `18446744073709551615`,
		},
		{
			data:     mustHexDecode("c249010000000000000000"),
			wantDiag: `18446744073709551616`,
		},
		{
			data:     mustHexDecode("3bffffffffffffffff"),
			wantDiag: `-18446744073709551616`,
		},
		{
			data:     mustHexDecode("c349010000000000000000"),
			wantDiag: `-18446744073709551617`,
		},
		{
			data:     mustHexDecode("20"),
			wantDiag: `-1`,
		},
		{
			data:     mustHexDecode("29"),
			wantDiag: `-10`,
		},
		{
			data:     mustHexDecode("3863"),
			wantDiag: `-100`,
		},
		{
			data:     mustHexDecode("3903e7"),
			wantDiag: `-1000`,
		},
		{
			data:     mustHexDecode("f90000"),
			wantDiag: `0.0`,
		},
		{
			data:     mustHexDecode("f98000"),
			wantDiag: `-0.0`,
		},
		{
			data:     mustHexDecode("f93c00"),
			wantDiag: `1.0`,
		},
		{
			data:     mustHexDecode("fb3ff199999999999a"),
			wantDiag: `1.1`,
		},
		{
			data:     mustHexDecode("f93e00"),
			wantDiag: `1.5`,
		},
		{
			data:     mustHexDecode("f97bff"),
			wantDiag: `65504.0`,
		},
		{
			data:     mustHexDecode("fa47c35000"),
			wantDiag: `100000.0`,
		},
		{
			data:     mustHexDecode("fa7f7fffff"),
			wantDiag: `3.4028234663852886e+38`,
		},
		{
			data:     mustHexDecode("fb7e37e43c8800759c"),
			wantDiag: `1.0e+300`,
		},
		{
			data:     mustHexDecode("f90001"),
			wantDiag: `5.960464477539063e-8`,
		},
		{
			data:     mustHexDecode("f90400"),
			wantDiag: `0.00006103515625`,
		},
		{
			data:     mustHexDecode("f9c400"),
			wantDiag: `-4.0`,
		},
		{
			data:     mustHexDecode("fbc010666666666666"),
			wantDiag: `-4.1`,
		},
		{
			data:     mustHexDecode("f97c00"),
			wantDiag: `Infinity`,
		},
		{
			data:     mustHexDecode("f97e00"),
			wantDiag: `NaN`,
		},
		{
			data:     mustHexDecode("f9fc00"),
			wantDiag: `-Infinity`,
		},
		{
			data:     mustHexDecode("fa7f800000"),
			wantDiag: `Infinity`,
		},
		{
			data:     mustHexDecode("fa7fc00000"),
			wantDiag: `NaN`,
		},
		{
			data:     mustHexDecode("faff800000"),
			wantDiag: `-Infinity`,
		},
		{
			data:     mustHexDecode("fb7ff0000000000000"),
			wantDiag: `Infinity`,
		},
		{
			data:     mustHexDecode("fb7ff8000000000000"),
			wantDiag: `NaN`,
		},
		{
			data:     mustHexDecode("fbfff0000000000000"),
			wantDiag: `-Infinity`,
		},
		{
			data:     mustHexDecode("f4"),
			wantDiag: `false`,
		},
		{
			data:     mustHexDecode("f5"),
			wantDiag: `true`,
		},
		{
			data:     mustHexDecode("f6"),
			wantDiag: `null`,
		},
		{
			data:     mustHexDecode("f7"),
			wantDiag: `undefined`,
		},
		{
			data:     mustHexDecode("f0"),
			wantDiag: `simple(16)`,
		},
		{
			data:     mustHexDecode("f8ff"),
			wantDiag: `simple(255)`,
		},
		{
			data:     mustHexDecode("c074323031332d30332d32315432303a30343a30305a"),
			wantDiag: `0("2013-03-21T20:04:00Z")`,
		},
		{
			data:     mustHexDecode("c11a514b67b0"),
			wantDiag: `1(1363896240)`,
		},
		{
			data:     mustHexDecode("c1fb41d452d9ec200000"),
			wantDiag: `1(1363896240.5)`,
		},
		{
			data:     mustHexDecode("d74401020304"),
			wantDiag: `23(h'01020304')`,
		},
		{
			data:     mustHexDecode("d818456449455446"),
			wantDiag: `24(h'6449455446')`,
		},
		{
			data:     mustHexDecode("d82076687474703a2f2f7777772e6578616d706c652e636f6d"),
			wantDiag: `32("http://www.example.com")`,
		},
		{
			data:     mustHexDecode("40"),
			wantDiag: `h''`,
		},
		{
			data:     mustHexDecode("4401020304"),
			wantDiag: `h'01020304'`,
		},
		{
			data:     mustHexDecode("60"),
			wantDiag: `""`,
		},
		{
			data:     mustHexDecode("6161"),
			wantDiag: `"a"`,
		},
		{
			data:     mustHexDecode("6449455446"),
			wantDiag: `"IETF"`,
		},
		{
			data:     mustHexDecode("62225c"),
			wantDiag: `"\"\\"`,
		},
		{
			data:     mustHexDecode("62c3bc"),
			wantDiag: `"\u00fc"`,
		},
		{
			data:     mustHexDecode("63e6b0b4"),
			wantDiag: `"\u6c34"`,
		},
		{
			data:     mustHexDecode("64f0908591"),
			wantDiag: `"\ud800\udd51"`,
		},
		{
			data:     mustHexDecode("80"),
			wantDiag: `[]`,
		},
		{
			data:     mustHexDecode("83010203"),
			wantDiag: `[1, 2, 3]`,
		},
		{
			data:     mustHexDecode("8301820203820405"),
			wantDiag: `[1, [2, 3], [4, 5]]`,
		},
		{
			data:     mustHexDecode("98190102030405060708090a0b0c0d0e0f101112131415161718181819"),
			wantDiag: `[1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25]`,
		},
		{
			data:     mustHexDecode("a0"),
			wantDiag: `{}`,
		},
		{
			data:     mustHexDecode("a201020304"),
			wantDiag: `{1: 2, 3: 4}`,
		},
		{
			data:     mustHexDecode("a26161016162820203"),
			wantDiag: `{"a": 1, "b": [2, 3]}`,
		},
		{
			data:     mustHexDecode("826161a161626163"),
			wantDiag: `["a", {"b": "c"}]`,
		},
		{
			data:     mustHexDecode("a56161614161626142616361436164614461656145"),
			wantDiag: `{"a": "A", "b": "B", "c": "C", "d": "D", "e": "E"}`,
		},
		{
			data:     mustHexDecode("5f42010243030405ff"),
			wantDiag: `(_ h'0102', h'030405')`,
		},
		{
			data:     mustHexDecode("7f657374726561646d696e67ff"),
			wantDiag: `(_ "strea", "ming")`,
		},
		{
			data:     mustHexDecode("9fff"),
			wantDiag: `[_ ]`,
		},
		{
			data:     mustHexDecode("9f018202039f0405ffff"),
			wantDiag: `[_ 1, [2, 3], [_ 4, 5]]`,
		},
		{
			data:     mustHexDecode("9f01820203820405ff"),
			wantDiag: `[_ 1, [2, 3], [4, 5]]`,
		},
		{
			data:     mustHexDecode("83018202039f0405ff"),
			wantDiag: `[1, [2, 3], [_ 4, 5]]`,
		},
		{
			data:     mustHexDecode("83019f0203ff820405"),
			wantDiag: `[1, [_ 2, 3], [4, 5]]`,
		},
		{
			data:     mustHexDecode("9f0102030405060708090a0b0c0d0e0f101112131415161718181819ff"),
			wantDiag: `[_ 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25]`,
		},
		{
			data:     mustHexDecode("bf61610161629f0203ffff"),
			wantDiag: `{_ "a": 1, "b": [_ 2, 3]}`,
		},
		{
			data:     mustHexDecode("826161bf61626163ff"),
			wantDiag: `["a", {_ "b": "c"}]`,
		},
		{
			data:     mustHexDecode("bf6346756ef563416d7421ff"),
			wantDiag: `{_ "Fun": true, "Amt": -2}`,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Diagnostic %d", i), func(t *testing.T) {
			str, err := Diagnose(tc.data)
			if err != nil {
				t.Errorf("Diagnostic(0x%x) returned error %q", tc.data, err)
			} else if str != tc.wantDiag {
				t.Errorf("Diagnostic(0x%x) returned `%s`, want `%s`", tc.data, str, tc.wantDiag)
			}

			str, rest, err := DiagnoseFirst(tc.data)
			if err != nil {
				t.Errorf("Diagnostic(0x%x) returned error %q", tc.data, err)
			} else if str != tc.wantDiag {
				t.Errorf("Diagnostic(0x%x) returned `%s`, want `%s`", tc.data, str, tc.wantDiag)
			}

			if rest == nil {
				t.Errorf("Diagnostic(0x%x) returned nil rest", tc.data)
			} else if len(rest) != 0 {
				t.Errorf("Diagnostic(0x%x) returned non-empty rest '%x'", tc.data, rest)
			}
		})
	}
}

func TestDiagnoseByteString(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		wantDiag string
		opts     *DiagOptions
	}{
		{
			name:     "base16",
			data:     mustHexDecode("4412345678"),
			wantDiag: `h'12345678'`,
			opts: &DiagOptions{
				ByteStringEncoding: ByteStringBase16Encoding,
			},
		},
		{
			name:     "base32",
			data:     mustHexDecode("4412345678"),
			wantDiag: `b32'CI2FM6A'`,
			opts: &DiagOptions{
				ByteStringEncoding: ByteStringBase32Encoding,
			},
		},
		{
			name:     "base32hex",
			data:     mustHexDecode("4412345678"),
			wantDiag: `h32'28Q5CU0'`,
			opts: &DiagOptions{
				ByteStringEncoding: ByteStringBase32HexEncoding,
			},
		},
		{
			name:     "base64",
			data:     mustHexDecode("4412345678"),
			wantDiag: `b64'EjRWeA'`,
			opts: &DiagOptions{
				ByteStringEncoding: ByteStringBase64Encoding,
			},
		},
		{
			name:     "without ByteStringHexWhitespace option",
			data:     mustHexDecode("4b48656c6c6f20776f726c64"),
			wantDiag: `h'48656c6c6f20776f726c64'`,
			opts: &DiagOptions{
				ByteStringHexWhitespace: false,
			},
		},
		{
			name:     "with ByteStringHexWhitespace option",
			data:     mustHexDecode("4b48656c6c6f20776f726c64"),
			wantDiag: `h'48 65 6c 6c 6f 20 77 6f 72 6c 64'`,
			opts: &DiagOptions{
				ByteStringHexWhitespace: true,
			},
		},
		{
			name:     "without ByteStringText option",
			data:     mustHexDecode("4b68656c6c6f20776f726c64"),
			wantDiag: `h'68656c6c6f20776f726c64'`,
			opts: &DiagOptions{
				ByteStringText: false,
			},
		},
		{
			name:     "with ByteStringText option",
			data:     mustHexDecode("4b68656c6c6f20776f726c64"),
			wantDiag: `'hello world'`,
			opts: &DiagOptions{
				ByteStringText: true,
			},
		},
		{
			name:     "without ByteStringText option and with ByteStringHexWhitespace option",
			data:     mustHexDecode("4b68656c6c6f20776f726c64"),
			wantDiag: `h'68 65 6c 6c 6f 20 77 6f 72 6c 64'`,
			opts: &DiagOptions{
				ByteStringText:          false,
				ByteStringHexWhitespace: true,
			},
		},
		{
			name:     "without ByteStringEmbeddedCBOR",
			data:     mustHexDecode("4101"),
			wantDiag: `h'01'`,
			opts: &DiagOptions{
				ByteStringEmbeddedCBOR: false,
			},
		},
		{
			name:     "with ByteStringEmbeddedCBOR",
			data:     mustHexDecode("4101"),
			wantDiag: `<<1>>`,
			opts: &DiagOptions{
				ByteStringEmbeddedCBOR: true,
			},
		},
		{
			name:     "multi CBOR items without ByteStringEmbeddedCBOR",
			data:     mustHexDecode("420102"),
			wantDiag: `h'0102'`,
			opts: &DiagOptions{
				ByteStringEmbeddedCBOR: false,
			},
		},
		{
			name:     "multi CBOR items with ByteStringEmbeddedCBOR",
			data:     mustHexDecode("420102"),
			wantDiag: `<<1, 2>>`,
			opts: &DiagOptions{
				ByteStringEmbeddedCBOR: true,
			},
		},
		{
			name:     "multi CBOR items with ByteStringEmbeddedCBOR",
			data:     mustHexDecode("4563666F6FF6"),
			wantDiag: `h'63666f6ff6'`,
			opts: &DiagOptions{
				ByteStringEmbeddedCBOR: false,
			},
		},
		{
			name:     "multi CBOR items with ByteStringEmbeddedCBOR",
			data:     mustHexDecode("4563666F6FF6"),
			wantDiag: `<<"foo", null>>`,
			opts: &DiagOptions{
				ByteStringEmbeddedCBOR: true,
			},
		},
		{
			name:     "indefinite-length byte string with no chunks",
			data:     mustHexDecode("5fff"),
			wantDiag: `''_`,
			opts:     &DiagOptions{},
		},
		{
			name:     "indefinite-length byte string with an empty byte string",
			data:     mustHexDecode("5f40ff"),
			wantDiag: `(_ h'')`, // RFC 8949, Section 8.1 says `(_ '')` but it looks wrong and conflicts with Appendix A.
			opts:     &DiagOptions{},
		},
		{
			name:     "indefinite-length byte string with two empty byte strings",
			data:     mustHexDecode("5f4040ff"),
			wantDiag: `(_ h'', h'')`,
			opts:     &DiagOptions{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dm, err := tc.opts.DiagMode()
			if err != nil {
				t.Errorf("DiagMode() for 0x%x returned error %q", tc.data, err)
			}

			str, err := dm.Diagnose(tc.data)
			if err != nil {
				t.Errorf("Diagnose(0x%x) returned error %q", tc.data, err)
			} else if str != tc.wantDiag {
				t.Errorf("Diagnose(0x%x) returned `%s`, want %s", tc.data, str, tc.wantDiag)
			}
		})
	}
}

func TestDiagnoseTextString(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		wantDiag string
		opts     *DiagOptions
	}{
		{
			name:     "\t",
			data:     mustHexDecode("6109"),
			wantDiag: `"\t"`,
			opts:     &DiagOptions{},
		},
		{
			name:     "\r",
			data:     mustHexDecode("610d"),
			wantDiag: `"\r"`,
			opts:     &DiagOptions{},
		},
		{
			name:     "other ascii",
			data:     mustHexDecode("611b"),
			wantDiag: `"\u001b"`,
			opts:     &DiagOptions{},
		},
		{
			name:     "valid UTF-8 text in byte string",
			data:     mustHexDecode("4d68656c6c6f2c20e4bda0e5a5bd"),
			wantDiag: `'hello, \u4f60\u597d'`,
			opts: &DiagOptions{
				ByteStringText: true,
			},
		},
		{
			name:     "valid UTF-8 text in text string",
			data:     mustHexDecode("6d68656c6c6f2c20e4bda0e5a5bd"),
			wantDiag: `"hello, \u4f60\u597d"`, // "hello, 你好"
			opts: &DiagOptions{
				ByteStringText: true,
			},
		},
		{
			name:     "invalid UTF-8 text in byte string",
			data:     mustHexDecode("4d68656c6c6fffeee4bda0e5a5bd"),
			wantDiag: `h'68656c6c6fffeee4bda0e5a5bd'`,
			opts: &DiagOptions{
				ByteStringText: true,
			},
		},
		{
			name:     "valid grapheme cluster text in byte string",
			data:     mustHexDecode("583448656c6c6f2c2027e29da4efb88fe2808df09f94a5270ae4bda0e5a5bdefbc8c22f09fa791e2808df09fa49de2808df09fa79122"),
			wantDiag: `'Hello, \'\u2764\ufe0f\u200d\ud83d\udd25\'\n\u4f60\u597d\uff0c"\ud83e\uddd1\u200d\ud83e\udd1d\u200d\ud83e\uddd1"'`,
			opts: &DiagOptions{
				ByteStringText: true,
			},
		},
		{
			name:     "valid grapheme cluster text in text string",
			data:     mustHexDecode("783448656c6c6f2c2027e29da4efb88fe2808df09f94a5270ae4bda0e5a5bdefbc8c22f09fa791e2808df09fa49de2808df09fa79122"),
			wantDiag: `"Hello, '\u2764\ufe0f\u200d\ud83d\udd25'\n\u4f60\u597d\uff0c\"\ud83e\uddd1\u200d\ud83e\udd1d\u200d\ud83e\uddd1\""`, // "Hello, '❤️‍🔥'\n你好，\"🧑‍🤝‍🧑\""
			opts: &DiagOptions{
				ByteStringText: true,
			},
		},
		{
			name:     "invalid grapheme cluster text in byte string",
			data:     mustHexDecode("583448656c6c6feeff27e29da4efb88fe2808df09f94a5270de4bda0e5a5bdefbc8c22f09fa791e2808df09fa49de2808df09fa79122"),
			wantDiag: `h'48656c6c6feeff27e29da4efb88fe2808df09f94a5270de4bda0e5a5bdefbc8c22f09fa791e2808df09fa49de2808df09fa79122'`,
			opts: &DiagOptions{
				ByteStringText: true,
			},
		},
		{
			name:     "valid U+FFFD replacement character in text string",
			data:     mustHexDecode("63efbfbd"),
			wantDiag: `"\ufffd"`,
			opts:     &DiagOptions{},
		},
		{
			name:     "indefinite-length text string with no chunks",
			data:     mustHexDecode("7fff"),
			wantDiag: `""_`,
			opts:     &DiagOptions{},
		},
		{
			name:     "indefinite-length text string with an empty text string",
			data:     mustHexDecode("7f60ff"),
			wantDiag: `(_ "")`,
			opts:     &DiagOptions{},
		},
		{
			name:     "indefinite-length text string with two empty text strings",
			data:     mustHexDecode("7f6060ff"),
			wantDiag: `(_ "", "")`,
			opts:     &DiagOptions{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dm, err := tc.opts.DiagMode()
			if err != nil {
				t.Errorf("DiagMode() for 0x%x returned error %q", tc.data, err)
			}

			str, err := dm.Diagnose(tc.data)
			if err != nil {
				t.Errorf("Diagnose(0x%x) returned error %q", tc.data, err)
			} else if str != tc.wantDiag {
				t.Errorf("Diagnose(0x%x) returned `%s`, want %s", tc.data, str, tc.wantDiag)
			}
		})
	}
}

func TestDiagnoseInvalidTextString(t *testing.T) {
	testCases := []struct {
		name         string
		data         []byte
		wantErrorMsg string
		opts         *DiagOptions
	}{
		{
			name:         "invalid UTF-8 text in text string",
			data:         mustHexDecode("6d68656c6c6fffeee4bda0e5a5bd"),
			wantErrorMsg: "invalid UTF-8 string",
			opts: &DiagOptions{
				ByteStringText: true,
			},
		},
		{
			name:         "invalid grapheme cluster text in text string",
			data:         mustHexDecode("783448656c6c6feeff27e29da4efb88fe2808df09f94a5270de4bda0e5a5bdefbc8c22f09fa791e2808df09fa49de2808df09fa79122"),
			wantErrorMsg: "invalid UTF-8 string",
			opts: &DiagOptions{
				ByteStringText: true,
			},
		},
		{
			name:         "invalid indefinite-length text string",
			data:         mustHexDecode("7f6040ff"),
			wantErrorMsg: `wrong element type`,
			opts: &DiagOptions{
				ByteStringText: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dm, err := tc.opts.DiagMode()
			if err != nil {
				t.Errorf("DiagMode() for 0x%x returned error %q", tc.data, err)
			}

			_, err = dm.Diagnose(tc.data)
			if err == nil {
				t.Errorf("Diagnose(0x%x) didn't return an error", tc.data)
			} else if !strings.Contains(err.Error(), tc.wantErrorMsg) {
				t.Errorf("Diagnose(0x%x) returned error %q", tc.data, err)
			}
		})
	}
}

func TestDiagnoseFloatingPointNumber(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		wantDiag string
		opts     *DiagOptions
	}{
		{
			name:     "float16 without FloatPrecisionIndicator option",
			data:     mustHexDecode("f93e00"),
			wantDiag: `1.5`,
			opts: &DiagOptions{
				FloatPrecisionIndicator: false,
			},
		},
		{
			name:     "float16 with FloatPrecisionIndicator option",
			data:     mustHexDecode("f93e00"),
			wantDiag: `1.5_1`,
			opts: &DiagOptions{
				FloatPrecisionIndicator: true,
			},
		},
		{
			name:     "float32 without FloatPrecisionIndicator option",
			data:     mustHexDecode("fa47c35000"),
			wantDiag: `100000.0`,
			opts: &DiagOptions{
				FloatPrecisionIndicator: false,
			},
		},
		{
			name:     "float32 with FloatPrecisionIndicator option",
			data:     mustHexDecode("fa47c35000"),
			wantDiag: `100000.0_2`,
			opts: &DiagOptions{
				FloatPrecisionIndicator: true,
			},
		},
		{
			name:     "float64 without FloatPrecisionIndicator option",
			data:     mustHexDecode("fbc010666666666666"),
			wantDiag: `-4.1`,
			opts: &DiagOptions{
				FloatPrecisionIndicator: false,
			},
		},
		{
			name:     "float64 with FloatPrecisionIndicator option",
			data:     mustHexDecode("fbc010666666666666"),
			wantDiag: `-4.1_3`,
			opts: &DiagOptions{
				FloatPrecisionIndicator: true,
			},
		},
		{
			name:     "with FloatPrecisionIndicator option",
			data:     mustHexDecode("c1fb41d452d9ec200000"),
			wantDiag: `1(1363896240.5_3)`,
			opts: &DiagOptions{
				FloatPrecisionIndicator: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dm, err := tc.opts.DiagMode()
			if err != nil {
				t.Errorf("DiagMode() for 0x%x returned error %q", tc.data, err)
			}

			str, err := dm.Diagnose(tc.data)
			if err != nil {
				t.Errorf("Diagnose(0x%x) returned error %q", tc.data, err)
			} else if str != tc.wantDiag {
				t.Errorf("Diagnose(0x%x) returned `%s`, want %s", tc.data, str, tc.wantDiag)
			}
		})
	}
}

func TestDiagnoseFirst(t *testing.T) {
	testCases := []struct {
		name         string
		data         []byte
		wantDiag     string
		wantRest     []byte
		wantErrorMsg string
	}{
		{
			name:         "with no trailing data",
			data:         mustHexDecode("f93e00"),
			wantDiag:     `1.5`,
			wantRest:     []byte{},
			wantErrorMsg: "",
		},
		{
			name:         "with CBOR Sequences",
			data:         mustHexDecode("f93e0064494554464401020304"),
			wantDiag:     `1.5`,
			wantRest:     mustHexDecode("64494554464401020304"),
			wantErrorMsg: "",
		},
		{
			name:         "with invalid CBOR trailing data",
			data:         mustHexDecode("f93e00ff494554464401020304"),
			wantDiag:     `1.5`,
			wantRest:     mustHexDecode("ff494554464401020304"),
			wantErrorMsg: "",
		},
		{
			name:         "with invalid CBOR data",
			data:         mustHexDecode("f93e"),
			wantDiag:     ``,
			wantRest:     nil,
			wantErrorMsg: "unexpected EOF",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			str, rest, err := DiagnoseFirst(tc.data)
			if str != tc.wantDiag {
				t.Errorf("DiagnoseFirst(0x%x) returned `%s`, want %s", tc.data, str, tc.wantDiag)
			}

			if bytes.Equal(rest, tc.wantRest) == false {
				if str != tc.wantDiag {
					t.Errorf("DiagnoseFirst(0x%x) returned rest `%x`, want rest %x", tc.data, rest, tc.wantRest)
				}
			}

			switch {
			case tc.wantErrorMsg == "" && err != nil:
				t.Errorf("DiagnoseFirst(0x%x) returned error %q", tc.data, err)
			case tc.wantErrorMsg != "" && err == nil:
				t.Errorf("DiagnoseFirst(0x%x) returned nil error, want error %q", tc.data, err)
			case tc.wantErrorMsg != "" && !strings.Contains(err.Error(), tc.wantErrorMsg):
				t.Errorf("DiagnoseFirst(0x%x) returned error %q, want error %q", tc.data, err, tc.wantErrorMsg)
			}
		})
	}
}

func TestDiagnoseCBORSequences(t *testing.T) {
	testCases := []struct {
		name        string
		data        []byte
		wantDiag    string
		opts        *DiagOptions
		returnError bool
	}{
		{
			name:     "CBOR Sequences without CBORSequence option",
			data:     mustHexDecode("f93e0064494554464401020304"),
			wantDiag: ``,
			opts: &DiagOptions{
				CBORSequence: false,
			},
			returnError: true,
		},
		{
			name:     "CBOR Sequences with CBORSequence option",
			data:     mustHexDecode("f93e0064494554464401020304"),
			wantDiag: `1.5, "IETF", h'01020304'`,
			opts: &DiagOptions{
				CBORSequence: true,
			},
			returnError: false,
		},
		{
			name:     "CBOR Sequences with CBORSequence option",
			data:     mustHexDecode("0102"),
			wantDiag: `1, 2`,
			opts: &DiagOptions{
				CBORSequence: true,
			},
			returnError: false,
		},
		{
			name:     "CBOR Sequences with CBORSequence option",
			data:     mustHexDecode("63666F6FF6"),
			wantDiag: `"foo", null`,
			opts: &DiagOptions{
				CBORSequence: true,
			},
			returnError: false,
		},
		{
			name:     "partial/incomplete CBOR Sequences",
			data:     mustHexDecode("f93e00644945544644010203"),
			wantDiag: `1.5, "IETF"`,
			opts: &DiagOptions{
				CBORSequence: true,
			},
			returnError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dm, err := tc.opts.DiagMode()
			if err != nil {
				t.Errorf("DiagMode() for 0x%x returned error %q", tc.data, err)
			}

			str, err := dm.Diagnose(tc.data)
			if tc.returnError && err == nil {
				t.Errorf("Diagnose(0x%x) returned error %q", tc.data, err)
			} else if !tc.returnError && err != nil {
				t.Errorf("Diagnose(0x%x) returned error %q", tc.data, err)
			}

			if str != tc.wantDiag {
				t.Errorf("Diagnose(0x%x) returned `%s`, want %s", tc.data, str, tc.wantDiag)
			}
		})
	}
}

func TestDiagnoseTag(t *testing.T) {
	testCases := []struct {
		name        string
		data        []byte
		wantDiag    string
		opts        *DiagOptions
		returnError bool
	}{
		{
			name:        "CBOR tag number 2 with not well-formed encoded CBOR data item",
			data:        mustHexDecode("c201"),
			wantDiag:    ``,
			opts:        &DiagOptions{},
			returnError: true,
		},
		{
			name:        "CBOR tag number 3 with not well-formed encoded CBOR data item",
			data:        mustHexDecode("c301"),
			wantDiag:    ``,
			opts:        &DiagOptions{},
			returnError: true,
		},
		{
			name:        "CBOR tag number 2 with well-formed encoded CBOR data item",
			data:        mustHexDecode("c240"),
			wantDiag:    `0`,
			opts:        &DiagOptions{},
			returnError: false,
		},
		{
			name:        "CBOR tag number 3 with well-formed encoded CBOR data item",
			data:        mustHexDecode("c340"),
			wantDiag:    `-1`, // -1 - n
			opts:        &DiagOptions{},
			returnError: false,
		},
		{
			name:        "CBOR tag number 2 with well-formed encoded CBOR data item",
			data:        mustHexDecode("c249010000000000000000"),
			wantDiag:    `18446744073709551616`,
			opts:        &DiagOptions{},
			returnError: false,
		},
		{
			name:        "CBOR tag number 3 with well-formed encoded CBOR data item",
			data:        mustHexDecode("c349010000000000000000"),
			wantDiag:    `-18446744073709551617`, // -1 - n
			opts:        &DiagOptions{},
			returnError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dm, err := tc.opts.DiagMode()
			if err != nil {
				t.Errorf("DiagMode() for 0x%x returned error %q", tc.data, err)
			}

			str, err := dm.Diagnose(tc.data)
			if tc.returnError && err == nil {
				t.Errorf("Diagnose(0x%x) returned error %q", tc.data, err)
			} else if !tc.returnError && err != nil {
				t.Errorf("Diagnose(0x%x) returned error %q", tc.data, err)
			}

			if str != tc.wantDiag {
				t.Errorf("Diagnose(0x%x) returned `%s`, want %s", tc.data, str, tc.wantDiag)
			}
		})
	}
}

func TestDiagnoseOptions(t *testing.T) {
	opts := DiagOptions{
		ByteStringEncoding:      ByteStringBase32Encoding,
		ByteStringHexWhitespace: true,
		ByteStringText:          false,
		ByteStringEmbeddedCBOR:  true,
		CBORSequence:            false,
		FloatPrecisionIndicator: true,
		MaxNestedLevels:         100,
		MaxArrayElements:        101,
		MaxMapPairs:             102,
	}
	dm, err := opts.DiagMode()
	if err != nil {
		t.Errorf("DiagMode() returned an error %v", err)
	}
	opts2 := dm.DiagOptions()
	if !reflect.DeepEqual(opts, opts2) {
		t.Errorf("DiagOptions() returned wrong options %v, want %v", opts2, opts)
	}

	opts = DiagOptions{
		ByteStringEncoding:      ByteStringBase64Encoding,
		ByteStringHexWhitespace: false,
		ByteStringText:          true,
		ByteStringEmbeddedCBOR:  false,
		CBORSequence:            true,
		FloatPrecisionIndicator: false,
		MaxNestedLevels:         100,
		MaxArrayElements:        101,
		MaxMapPairs:             102,
	}
	dm, err = opts.DiagMode()
	if err != nil {
		t.Errorf("DiagMode() returned an error %v", err)
	}
	opts2 = dm.DiagOptions()
	if !reflect.DeepEqual(opts, opts2) {
		t.Errorf("DiagOptions() returned wrong options %v, want %v", opts2, opts)
	}
}

func TestInvalidDiagnoseOptions(t *testing.T) {
	opts := &DiagOptions{
		ByteStringEncoding: ByteStringBase64Encoding + 1,
	}
	_, err := opts.DiagMode()
	if err == nil {
		t.Errorf("DiagMode() with invalid ByteStringEncoding option didn't return an error")
	}
}

func TestDiagnoseExtraneousData(t *testing.T) {
	data := mustHexDecode("63666F6FF6")
	_, err := Diagnose(data)
	if err == nil {
		t.Errorf("Diagnose(0x%x) didn't return an error", data)
	} else if !strings.Contains(err.Error(), `extraneous data`) {
		t.Errorf("Diagnose(0x%x) returned error %q", data, err)
	}

	_, _, err = DiagnoseFirst(data)
	if err != nil {
		t.Errorf("DiagnoseFirst(0x%x) returned error %v", data, err)
	}
}

func TestDiagnoseNotwellformedData(t *testing.T) {
	data := mustHexDecode("5f4060ff")
	_, err := Diagnose(data)
	if err == nil {
		t.Errorf("Diagnose(0x%x) didn't return an error", data)
	} else if !strings.Contains(err.Error(), `wrong element type`) {
		t.Errorf("Diagnose(0x%x) returned error %q", data, err)
	}
}

func TestDiagnoseEmptyData(t *testing.T) {
	var emptyData []byte

	defaultMode, _ := DiagOptions{}.DiagMode()
	sequenceMode, _ := DiagOptions{CBORSequence: true}.DiagMode()

	testCases := []struct {
		name string
		dm   DiagMode
	}{
		{name: "default", dm: defaultMode},
		{name: "sequence", dm: sequenceMode},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := tc.dm.Diagnose(emptyData)
			if s != "" {
				t.Errorf("Diagnose() didn't return empty notation for empty data")
			}
			if err != io.EOF {
				t.Errorf("Diagnose() didn't return io.EOF for empty data")
			}

			s, rest, err := tc.dm.DiagnoseFirst(emptyData)
			if s != "" {
				t.Errorf("DiagnoseFirst() didn't return empty notation for empty data")
			}
			if len(rest) != 0 {
				t.Errorf("DiagnoseFirst() didn't return empty rest for empty data")
			}
			if err != io.EOF {
				t.Errorf("DiagnoseFirst() didn't return io.EOF for empty data")
			}
		})
	}
}

func TestDiagnoseInvalidByteStringEncoding(t *testing.T) {
	_, err := DiagOptions{
		ByteStringEncoding: maxByteStringEncoding,
	}.DiagMode()
	if err == nil {
		t.Fatal("DiagMode() expected error for invalid ByteStringEncoding, got nil")
	}
	if !strings.Contains(err.Error(), "invalid ByteStringEncoding") {
		t.Errorf("DiagMode() error = %q, want error containing \"invalid ByteStringEncoding\"", err.Error())
	}
}

func BenchmarkDiagnose(b *testing.B) {
	testCases := []struct {
		name  string
		opts  DiagOptions
		input []byte
	}{
		{
			name:  "escaped character in text string",
			opts:  DiagOptions{},
			input: mustHexDecode("62c3bc"), // "\u00fc"
		},
		{
			name:  "byte string base16 encoding",
			opts:  DiagOptions{ByteStringEncoding: ByteStringBase16Encoding},
			input: []byte("\x45hello"),
		},
		{
			name:  "byte string base32 encoding",
			opts:  DiagOptions{ByteStringEncoding: ByteStringBase32Encoding},
			input: []byte("\x45hello"),
		},
		{
			name:  "byte string base32hex encoding",
			opts:  DiagOptions{ByteStringEncoding: ByteStringBase32HexEncoding},
			input: []byte("\x45hello"),
		},
		{
			name:  "byte string base64url encoding",
			opts:  DiagOptions{ByteStringEncoding: ByteStringBase64Encoding},
			input: []byte("\x45hello"),
		},
	}
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			dm, err := tc.opts.DiagMode()
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = dm.Diagnose(tc.input)
			}
		})
	}
}
