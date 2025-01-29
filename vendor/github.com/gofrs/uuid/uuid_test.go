// Copyright (C) 2013-2018 by Maxim Bublis <b@codemonkey.ru>
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package uuid

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"
)

func TestUUID(t *testing.T) {
	t.Run("IsNil", testUUIDIsNil)
	t.Run("Bytes", testUUIDBytes)
	t.Run("String", testUUIDString)
	t.Run("Version", testUUIDVersion)
	t.Run("Variant", testUUIDVariant)
	t.Run("SetVersion", testUUIDSetVersion)
	t.Run("SetVariant", testUUIDSetVariant)
	t.Run("Format", testUUIDFormat)
}

func testUUIDIsNil(t *testing.T) {
	u := UUID{}
	got := u.IsNil()
	want := true
	if got != want {
		t.Errorf("%v.IsNil() = %t, want %t", u, got, want)
	}
}

func testUUIDBytes(t *testing.T) {
	got := codecTestUUID.Bytes()
	want := codecTestData
	if !bytes.Equal(got, want) {
		t.Errorf("%v.Bytes() = %x, want %x", codecTestUUID, got, want)
	}
}

func testUUIDString(t *testing.T) {
	got := NamespaceDNS.String()
	want := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	if got != want {
		t.Errorf("%v.String() = %q, want %q", NamespaceDNS, got, want)
	}
}

func testUUIDVersion(t *testing.T) {
	u := UUID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if got, want := u.Version(), V1; got != want {
		t.Errorf("%v.Version() == %d, want %d", u, got, want)
	}
}

func testUUIDVariant(t *testing.T) {
	tests := []struct {
		u    UUID
		want byte
	}{
		{
			u:    UUID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			want: VariantNCS,
		},
		{
			u:    UUID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			want: VariantRFC4122,
		},
		{
			u:    UUID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			want: VariantMicrosoft,
		},
		{
			u:    UUID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			want: VariantFuture,
		},
	}
	for _, tt := range tests {
		if got := tt.u.Variant(); got != tt.want {
			t.Errorf("%v.Variant() == %d, want %d", tt.u, got, tt.want)
		}
	}
}

func testUUIDSetVersion(t *testing.T) {
	u := UUID{}
	want := V4
	u.SetVersion(want)
	if got := u.Version(); got != want {
		t.Errorf("%v.Version() == %d after SetVersion(%d)", u, got, want)
	}
}

func testUUIDSetVariant(t *testing.T) {
	variants := []byte{
		VariantNCS,
		VariantRFC4122,
		VariantMicrosoft,
		VariantFuture,
	}
	for _, want := range variants {
		u := UUID{}
		u.SetVariant(want)
		if got := u.Variant(); got != want {
			t.Errorf("%v.Variant() == %d after SetVariant(%d)", u, got, want)
		}
	}
}

func testUUIDFormat(t *testing.T) {
	val := Must(FromString("12345678-90ab-cdef-1234-567890abcdef"))
	tests := []struct {
		u    UUID
		f    string
		want string
	}{
		{u: val, f: "%s", want: "12345678-90ab-cdef-1234-567890abcdef"},
		{u: val, f: "%S", want: "12345678-90AB-CDEF-1234-567890ABCDEF"},
		{u: val, f: "%q", want: `"12345678-90ab-cdef-1234-567890abcdef"`},
		{u: val, f: "%x", want: "1234567890abcdef1234567890abcdef"},
		{u: val, f: "%X", want: "1234567890ABCDEF1234567890ABCDEF"},
		{u: val, f: "%v", want: "12345678-90ab-cdef-1234-567890abcdef"},
		{u: val, f: "%+v", want: "12345678-90ab-cdef-1234-567890abcdef"},
		{u: val, f: "%#v", want: "[16]uint8{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}"},
		{u: val, f: "%T", want: "uuid.UUID"},
		{u: val, f: "%t", want: "%!t(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
		{u: val, f: "%b", want: "%!b(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
		{u: val, f: "%c", want: "%!c(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
		{u: val, f: "%d", want: "%!d(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
		{u: val, f: "%e", want: "%!e(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
		{u: val, f: "%E", want: "%!E(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
		{u: val, f: "%f", want: "%!f(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
		{u: val, f: "%F", want: "%!F(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
		{u: val, f: "%g", want: "%!g(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
		{u: val, f: "%G", want: "%!G(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
		{u: val, f: "%o", want: "%!o(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
		{u: val, f: "%U", want: "%!U(uuid.UUID=12345678-90ab-cdef-1234-567890abcdef)"},
	}
	for _, tt := range tests {
		got := fmt.Sprintf(tt.f, tt.u)
		if tt.want != got {
			t.Errorf(`Format("%s") got %s, want %s`, tt.f, got, tt.want)
		}
	}
}

func TestMust(t *testing.T) {
	sentinel := fmt.Errorf("uuid: sentinel error")
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("did not panic, want %v", sentinel)
		}
		err, ok := r.(error)
		if !ok {
			t.Fatalf("panicked with %T, want error (%v)", r, sentinel)
		}
		if err != sentinel {
			t.Fatalf("panicked with %v, want %v", err, sentinel)
		}
	}()
	fn := func() (UUID, error) {
		return Nil, sentinel
	}
	Must(fn())
}

func TestTimeFromTimestamp(t *testing.T) {
	tests := []struct {
		t    Timestamp
		want time.Time
	}{
		// a zero timestamp represents October 15, 1582 at midnight UTC
		{t: Timestamp(0), want: time.Date(1582, 10, 15, 0, 0, 0, 0, time.UTC)},
		// a one value is 100ns later
		{t: Timestamp(1), want: time.Date(1582, 10, 15, 0, 0, 0, 100, time.UTC)},
		// 10 million 100ns intervals later is one second
		{t: Timestamp(10000000), want: time.Date(1582, 10, 15, 0, 0, 1, 0, time.UTC)},
		{t: Timestamp(60 * 10000000), want: time.Date(1582, 10, 15, 0, 1, 0, 0, time.UTC)},
		{t: Timestamp(60 * 60 * 10000000), want: time.Date(1582, 10, 15, 1, 0, 0, 0, time.UTC)},
		{t: Timestamp(24 * 60 * 60 * 10000000), want: time.Date(1582, 10, 16, 0, 0, 0, 0, time.UTC)},
		{t: Timestamp(365 * 24 * 60 * 60 * 10000000), want: time.Date(1583, 10, 15, 0, 0, 0, 0, time.UTC)},
		// maximum timestamp value in a UUID is the year 5236
		{t: Timestamp(uint64(1<<60 - 1)), want: time.Date(5236, 03, 31, 21, 21, 0, 684697500, time.UTC)},
	}
	for _, tt := range tests {
		got, _ := tt.t.Time()
		if !got.Equal(tt.want) {
			t.Errorf("%v.Time() == %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestTimestampFromV1(t *testing.T) {
	tests := []struct {
		u       UUID
		want    Timestamp
		wanterr bool
	}{
		{u: Must(NewV4()), wanterr: true},
		{u: Must(FromString("00000000-0000-1000-0000-000000000000")), want: 0},
		{u: Must(FromString("424f137e-a2aa-11e8-98d0-529269fb1459")), want: 137538640775418750},
		{u: Must(FromString("ffffffff-ffff-1fff-ffff-ffffffffffff")), want: Timestamp(1<<60 - 1)},
	}
	for _, tt := range tests {
		got, goterr := TimestampFromV1(tt.u)
		if tt.wanterr && goterr == nil {
			t.Errorf("TimestampFromV1(%v) want error, got %v", tt.u, got)
		} else if tt.want != got {
			t.Errorf("TimestampFromV1(%v) got %v, want %v", tt.u, got, tt.want)
		}
	}
}

func TestTimestampFromV6(t *testing.T) {
	tests := []struct {
		u       UUID
		want    Timestamp
		wanterr bool
	}{
		{u: Must(NewV1()), wanterr: true},
		{u: Must(FromString("00000000-0000-6000-0000-000000000000")), want: 0},
		{u: Must(FromString("1ec06cff-e9b1-621c-8627-ba3fd7e551c9")), want: 138493178941215260},
		{u: Must(FromString("ffffffff-ffff-6fff-ffff-ffffffffffff")), want: Timestamp(1<<60 - 1)},
	}

	for _, tt := range tests {
		got, err := TimestampFromV6(tt.u)

		switch {
		case tt.wanterr && err == nil:
			t.Errorf("TimestampFromV6(%v) want error, got %v", tt.u, got)

		case tt.want != got:
			t.Errorf("TimestampFromV6(%v) got %v, want %v", tt.u, got, tt.want)
		}
	}
}

func BenchmarkFormat(b *testing.B) {
	var tests = []string{
		"%s",
		"%S",
		"%q",
		"%x",
		"%X",
		"%v",
		"%+v",
		"%#v",
	}
	for _, x := range tests {
		b.Run(x[1:], func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				fmt.Fprintf(io.Discard, x, &codecTestUUID)
			}
		})
	}
}
