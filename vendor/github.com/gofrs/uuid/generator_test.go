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
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestGenerator(t *testing.T) {
	t.Run("NewV1", testNewV1)
	t.Run("NewV3", testNewV3)
	t.Run("NewV4", testNewV4)
	t.Run("NewV5", testNewV5)
	t.Run("NewV6", testNewV6)
	t.Run("NewV7", testNewV7)
}

func testNewV1(t *testing.T) {
	t.Run("Basic", testNewV1Basic)
	t.Run("BasicWithOptions", testNewV1BasicWithOptions)
	t.Run("DifferentAcrossCalls", testNewV1DifferentAcrossCalls)
	t.Run("StaleEpoch", testNewV1StaleEpoch)
	t.Run("FaultyRand", testNewV1FaultyRand)
	t.Run("FaultyRandWithOptions", testNewV1FaultyRandWithOptions)
	t.Run("MissingNetwork", testNewV1MissingNetwork)
	t.Run("MissingNetworkWithOptions", testNewV1MissingNetworkWithOptions)
	t.Run("MissingNetworkFaultyRand", testNewV1MissingNetworkFaultyRand)
	t.Run("MissingNetworkFaultyRandWithOptions", testNewV1MissingNetworkFaultyRandWithOptions)
}

func TestNewGenWithHWAF(t *testing.T) {
	addr := []byte{0, 1, 2, 3, 4, 42}

	fn := func() (net.HardwareAddr, error) {
		return addr, nil
	}

	var g *Gen
	var err error
	var uuid UUID

	g = NewGenWithHWAF(fn)

	if g == nil {
		t.Fatal("g is unexpectedly nil")
	}

	uuid, err = g.NewV1()
	if err != nil {
		t.Fatalf("g.NewV1() err = %v, want <nil>", err)
	}

	node := uuid[10:]

	if !bytes.Equal(addr, node) {
		t.Fatalf("node = %v, want %v", node, addr)
	}
}

func testNewV1Basic(t *testing.T) {
	u, err := NewV1()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := u.Version(), V1; got != want {
		t.Errorf("generated UUID with version %d, want %d", got, want)
	}
	if got, want := u.Variant(), VariantRFC4122; got != want {
		t.Errorf("generated UUID with variant %d, want %d", got, want)
	}
}

func testNewV1BasicWithOptions(t *testing.T) {
	g := NewGenWithOptions(
		WithHWAddrFunc(nil),
		WithEpochFunc(nil),
		WithRandomReader(nil),
	)
	u, err := g.NewV1()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := u.Version(), V1; got != want {
		t.Errorf("generated UUID with version %d, want %d", got, want)
	}
	if got, want := u.Variant(), VariantRFC4122; got != want {
		t.Errorf("generated UUID with variant %d, want %d", got, want)
	}
}

func testNewV1DifferentAcrossCalls(t *testing.T) {
	u1, err := NewV1()
	if err != nil {
		t.Fatal(err)
	}
	u2, err := NewV1()
	if err != nil {
		t.Fatal(err)
	}
	if u1 == u2 {
		t.Errorf("generated identical UUIDs across calls: %v", u1)
	}
}

func testNewV1StaleEpoch(t *testing.T) {
	g := &Gen{
		epochFunc: func() time.Time {
			return time.Unix(0, 0)
		},
		hwAddrFunc: defaultHWAddrFunc,
		rand:       rand.Reader,
	}
	u1, err := g.NewV1()
	if err != nil {
		t.Fatal(err)
	}
	u2, err := g.NewV1()
	if err != nil {
		t.Fatal(err)
	}
	if u1 == u2 {
		t.Errorf("generated identical UUIDs across calls: %v", u1)
	}
}

func testNewV1FaultyRand(t *testing.T) {
	g := &Gen{
		epochFunc:  time.Now,
		hwAddrFunc: defaultHWAddrFunc,
		rand: &faultyReader{
			readToFail: 0, // fail immediately
		},
	}
	u, err := g.NewV1()
	if err == nil {
		t.Fatalf("got %v, want error", u)
	}
	if u != Nil {
		t.Fatalf("got %v on error, want Nil", u)
	}
}

func testNewV1MissingNetwork(t *testing.T) {
	g := &Gen{
		epochFunc: time.Now,
		hwAddrFunc: func() (net.HardwareAddr, error) {
			return []byte{}, fmt.Errorf("uuid: no hw address found")
		},
		rand: rand.Reader,
	}
	_, err := g.NewV1()
	if err != nil {
		t.Errorf("did not handle missing network interfaces: %v", err)
	}
}

func testNewV1MissingNetworkWithOptions(t *testing.T) {
	g := NewGenWithOptions(
		WithHWAddrFunc(func() (net.HardwareAddr, error) {
			return []byte{}, fmt.Errorf("uuid: no hw address found")
		}),
	)
	_, err := g.NewV1()
	if err != nil {
		t.Errorf("did not handle missing network interfaces: %v", err)
	}
}

func testNewV1MissingNetworkFaultyRand(t *testing.T) {
	g := &Gen{
		epochFunc: time.Now,
		hwAddrFunc: func() (net.HardwareAddr, error) {
			return []byte{}, fmt.Errorf("uuid: no hw address found")
		},
		rand: &faultyReader{
			readToFail: 1,
		},
	}
	u, err := g.NewV1()
	if err == nil {
		t.Errorf("did not error on faulty reader and missing network, got %v", u)
	}
}

func testNewV1MissingNetworkFaultyRandWithOptions(t *testing.T) {
	g := NewGenWithOptions(
		WithHWAddrFunc(func() (net.HardwareAddr, error) {
			return []byte{}, fmt.Errorf("uuid: no hw address found")
		}),
		WithRandomReader(&faultyReader{
			readToFail: 1,
		}),
	)

	u, err := g.NewV1()
	if err == nil {
		t.Errorf("did not error on faulty reader and missing network, got %v", u)
	}
}

func testNewV1FaultyRandWithOptions(t *testing.T) {
	g := NewGenWithOptions(WithRandomReader(&faultyReader{
		readToFail: 0, // fail immediately
	}),
	)
	u, err := g.NewV1()
	if err == nil {
		t.Errorf("did not error on faulty reader and missing network, got %v", u)
	}
}

func testNewV3(t *testing.T) {
	t.Run("Basic", testNewV3Basic)
	t.Run("EqualNames", testNewV3EqualNames)
	t.Run("DifferentNamespaces", testNewV3DifferentNamespaces)
}

func testNewV3Basic(t *testing.T) {
	ns := NamespaceDNS
	name := "www.example.com"
	u := NewV3(ns, name)
	if got, want := u.Version(), V3; got != want {
		t.Errorf("NewV3(%v, %q): got version %d, want %d", ns, name, got, want)
	}
	if got, want := u.Variant(), VariantRFC4122; got != want {
		t.Errorf("NewV3(%v, %q): got variant %d, want %d", ns, name, got, want)
	}
	want := "5df41881-3aed-3515-88a7-2f4a814cf09e"
	if got := u.String(); got != want {
		t.Errorf("NewV3(%v, %q) = %q, want %q", ns, name, got, want)
	}
}

func testNewV3EqualNames(t *testing.T) {
	ns := NamespaceDNS
	name := "example.com"
	u1 := NewV3(ns, name)
	u2 := NewV3(ns, name)
	if u1 != u2 {
		t.Errorf("NewV3(%v, %q) generated %v and %v across two calls", ns, name, u1, u2)
	}
}

func testNewV3DifferentNamespaces(t *testing.T) {
	name := "example.com"
	ns1 := NamespaceDNS
	ns2 := NamespaceURL
	u1 := NewV3(ns1, name)
	u2 := NewV3(ns2, name)
	if u1 == u2 {
		t.Errorf("NewV3(%v, %q) == NewV3(%d, %q) (%v)", ns1, name, ns2, name, u1)
	}
}

func testNewV4(t *testing.T) {
	t.Run("Basic", testNewV4Basic)
	t.Run("DifferentAcrossCalls", testNewV4DifferentAcrossCalls)
	t.Run("FaultyRand", testNewV4FaultyRand)
	t.Run("FaultyRandWithOptions", testNewV4FaultyRandWithOptions)
	t.Run("ShortRandomRead", testNewV4ShortRandomRead)
	t.Run("ShortRandomReadWithOptions", testNewV4ShortRandomReadWithOptions)
}

func testNewV4Basic(t *testing.T) {
	u, err := NewV4()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := u.Version(), V4; got != want {
		t.Errorf("got version %d, want %d", got, want)
	}
	if got, want := u.Variant(), VariantRFC4122; got != want {
		t.Errorf("got variant %d, want %d", got, want)
	}
}

func testNewV4DifferentAcrossCalls(t *testing.T) {
	u1, err := NewV4()
	if err != nil {
		t.Fatal(err)
	}
	u2, err := NewV4()
	if err != nil {
		t.Fatal(err)
	}
	if u1 == u2 {
		t.Errorf("generated identical UUIDs across calls: %v", u1)
	}
}

func testNewV4FaultyRand(t *testing.T) {
	g := &Gen{
		epochFunc:  time.Now,
		hwAddrFunc: defaultHWAddrFunc,
		rand: &faultyReader{
			readToFail: 0, // fail immediately
		},
	}
	u, err := g.NewV4()
	if err == nil {
		t.Errorf("got %v, nil error", u)
	}
}

func testNewV4FaultyRandWithOptions(t *testing.T) {
	g := NewGenWithOptions(
		WithRandomReader(&faultyReader{
			readToFail: 0, // fail immediately
		}),
	)
	u, err := g.NewV4()
	if err == nil {
		t.Errorf("got %v, nil error", u)
	}
}

func testNewV4ShortRandomRead(t *testing.T) {
	g := &Gen{
		epochFunc: time.Now,
		hwAddrFunc: func() (net.HardwareAddr, error) {
			return []byte{}, fmt.Errorf("uuid: no hw address found")
		},
		rand: bytes.NewReader([]byte{42}),
	}
	u, err := g.NewV4()
	if err == nil {
		t.Errorf("got %v, nil error", u)
	}
}

func testNewV4ShortRandomReadWithOptions(t *testing.T) {
	g := NewGenWithOptions(
		WithHWAddrFunc(func() (net.HardwareAddr, error) {
			return []byte{}, fmt.Errorf("uuid: no hw address found")
		}),
		WithRandomReader(&faultyReader{
			readToFail: 0, // fail immediately
		}),
	)
	u, err := g.NewV4()
	if err == nil {
		t.Errorf("got %v, nil error", u)
	}
}

func testNewV5(t *testing.T) {
	t.Run("Basic", testNewV5Basic)
	t.Run("EqualNames", testNewV5EqualNames)
	t.Run("DifferentNamespaces", testNewV5DifferentNamespaces)
}

func testNewV5Basic(t *testing.T) {
	ns := NamespaceDNS
	name := "www.example.com"
	u := NewV5(ns, name)
	if got, want := u.Version(), V5; got != want {
		t.Errorf("NewV5(%v, %q): got version %d, want %d", ns, name, got, want)
	}
	if got, want := u.Variant(), VariantRFC4122; got != want {
		t.Errorf("NewV5(%v, %q): got variant %d, want %d", ns, name, got, want)
	}
	want := "2ed6657d-e927-568b-95e1-2665a8aea6a2"
	if got := u.String(); got != want {
		t.Errorf("NewV5(%v, %q) = %q, want %q", ns, name, got, want)
	}
}

func testNewV5EqualNames(t *testing.T) {
	ns := NamespaceDNS
	name := "example.com"
	u1 := NewV5(ns, name)
	u2 := NewV5(ns, name)
	if u1 != u2 {
		t.Errorf("NewV5(%v, %q) generated %v and %v across two calls", ns, name, u1, u2)
	}
}

func testNewV5DifferentNamespaces(t *testing.T) {
	name := "example.com"
	ns1 := NamespaceDNS
	ns2 := NamespaceURL
	u1 := NewV5(ns1, name)
	u2 := NewV5(ns2, name)
	if u1 == u2 {
		t.Errorf("NewV5(%v, %q) == NewV5(%v, %q) (%v)", ns1, name, ns2, name, u1)
	}
}

func testNewV6(t *testing.T) {
	t.Run("Basic", testNewV6Basic)
	t.Run("DifferentAcrossCalls", testNewV6DifferentAcrossCalls)
	t.Run("StaleEpoch", testNewV6StaleEpoch)
	t.Run("StaleEpochWithOptions", testNewV6StaleEpochWithOptions)
	t.Run("FaultyRand", testNewV6FaultyRand)
	t.Run("FaultyRandWithOptions", testNewV6FaultyRandWithOptions)
	t.Run("ShortRandomRead", testNewV6ShortRandomRead)
	t.Run("ShortRandomReadWithOptions", testNewV6ShortRandomReadWithOptions)
	t.Run("KSortable", testNewV6KSortable)
}

func testNewV6Basic(t *testing.T) {
	u, err := NewV6()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := u.Version(), V6; got != want {
		t.Errorf("generated UUID with version %d, want %d", got, want)
	}
	if got, want := u.Variant(), VariantRFC4122; got != want {
		t.Errorf("generated UUID with variant %d, want %d", got, want)
	}
}

func testNewV6DifferentAcrossCalls(t *testing.T) {
	u1, err := NewV6()
	if err != nil {
		t.Fatal(err)
	}
	u2, err := NewV6()
	if err != nil {
		t.Fatal(err)
	}
	if u1 == u2 {
		t.Errorf("generated identical UUIDs across calls: %v", u1)
	}
}

func testNewV6StaleEpoch(t *testing.T) {
	g := &Gen{
		epochFunc: func() time.Time {
			return time.Unix(0, 0)
		},
		hwAddrFunc: defaultHWAddrFunc,
		rand:       rand.Reader,
	}
	u1, err := g.NewV6()
	if err != nil {
		t.Fatal(err)
	}
	u2, err := g.NewV6()
	if err != nil {
		t.Fatal(err)
	}
	if u1 == u2 {
		t.Errorf("generated identical UUIDs across calls: %v", u1)
	}
}

func testNewV6StaleEpochWithOptions(t *testing.T) {
	g := NewGenWithOptions(
		WithEpochFunc(func() time.Time {
			return time.Unix(0, 0)
		}),
	)
	u1, err := g.NewV6()
	if err != nil {
		t.Fatal(err)
	}
	u2, err := g.NewV6()
	if err != nil {
		t.Fatal(err)
	}
	if u1 == u2 {
		t.Errorf("generated identical UUIDs across calls: %v", u1)
	}
}

func testNewV6FaultyRand(t *testing.T) {
	t.Run("randomData", func(t *testing.T) {
		g := &Gen{
			epochFunc:  time.Now,
			hwAddrFunc: defaultHWAddrFunc,
			rand: &faultyReader{
				readToFail: 0, // fail immediately
			},
		}
		u, err := g.NewV6()
		if err == nil {
			t.Fatalf("got %v, want error", u)
		}
		if u != Nil {
			t.Fatalf("got %v on error, want Nil", u)
		}
	})

	t.Run("clockSequence", func(t *testing.T) {
		g := &Gen{
			epochFunc:  time.Now,
			hwAddrFunc: defaultHWAddrFunc,
			rand: &faultyReader{
				readToFail: 1, // fail immediately
			},
		}
		u, err := g.NewV6()
		if err == nil {
			t.Fatalf("got %v, want error", u)
		}
		if u != Nil {
			t.Fatalf("got %v on error, want Nil", u)
		}
	})
}

func testNewV6FaultyRandWithOptions(t *testing.T) {
	t.Run("randomData", func(t *testing.T) {
		g := NewGenWithOptions(
			WithRandomReader(&faultyReader{
				readToFail: 0, // fail immediately
			}),
		)
		u, err := g.NewV6()
		if err == nil {
			t.Fatalf("got %v, want error", u)
		}
		if u != Nil {
			t.Fatalf("got %v on error, want Nil", u)
		}
	})

	t.Run("clockSequence", func(t *testing.T) {
		g := NewGenWithOptions(
			WithRandomReader(&faultyReader{
				readToFail: 1, // fail immediately
			}),
		)
		u, err := g.NewV6()
		if err == nil {
			t.Fatalf("got %v, want error", u)
		}
		if u != Nil {
			t.Fatalf("got %v on error, want Nil", u)
		}
	})
}

func testNewV6ShortRandomRead(t *testing.T) {
	g := &Gen{
		epochFunc: time.Now,
		rand:      bytes.NewReader([]byte{42}),
	}
	u, err := g.NewV6()
	if err == nil {
		t.Errorf("got %v, nil error", u)
	}
}

func testNewV6ShortRandomReadWithOptions(t *testing.T) {
	g := NewGenWithOptions(
		WithRandomReader(bytes.NewReader([]byte{42})),
	)
	u, err := g.NewV6()
	if err == nil {
		t.Errorf("got %v, nil error", u)
	}
}

func testNewV6KSortable(t *testing.T) {
	uuids := make([]UUID, 10)
	for i := range uuids {
		u, err := NewV6()
		testErrCheck(t, "NewV6()", "", err)

		uuids[i] = u

		time.Sleep(time.Microsecond)
	}

	for i := 1; i < len(uuids); i++ {
		p, n := uuids[i-1], uuids[i]
		isLess := p.String() < n.String()
		if !isLess {
			t.Errorf("uuids[%d] (%s) not less than uuids[%d] (%s)", i-1, p, i, n)
		}
	}
}

func testNewV7(t *testing.T) {
	t.Run("Basic", makeTestNewV7Basic())
	t.Run("TestVector", makeTestNewV7TestVector())
	t.Run("Basic10000000", makeTestNewV7Basic10000000())
	t.Run("DifferentAcrossCalls", makeTestNewV7DifferentAcrossCalls())
	t.Run("StaleEpoch", makeTestNewV7StaleEpoch())
	t.Run("StaleEpochWithOptions", makeTestNewV7StaleEpochWithOptions())
	t.Run("FaultyRand", makeTestNewV7FaultyRand())
	t.Run("FaultyRandWithOptions", makeTestNewV7FaultyRandWithOptions())
	t.Run("ShortRandomRead", makeTestNewV7ShortRandomRead())
	t.Run("KSortable", makeTestNewV7KSortable())
	t.Run("ClockSequence", makeTestNewV7ClockSequence())
}

func makeTestNewV7Basic() func(t *testing.T) {
	return func(t *testing.T) {
		u, err := NewV7()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := u.Version(), V7; got != want {
			t.Errorf("got version %d, want %d", got, want)
		}
		if got, want := u.Variant(), VariantRFC4122; got != want {
			t.Errorf("got variant %d, want %d", got, want)
		}
	}
}

// makeTestNewV7TestVector as defined in Draft04
func makeTestNewV7TestVector() func(t *testing.T) {
	return func(t *testing.T) {
		pRand := make([]byte, 10)
		//first 2 bytes will be read by clockSeq. First 4 bits will be overridden by Version. The next bits should be 0xCC3(3267)
		binary.LittleEndian.PutUint16(pRand[:2], uint16(0xCC3))
		//8bytes will be read for rand_b. First 2 bits will be overridden by Variant
		binary.LittleEndian.PutUint64(pRand[2:], uint64(0x18C4DC0C0C07398F))

		g := &Gen{
			epochFunc: func() time.Time {
				return time.UnixMilli(1645557742000)
			},
			rand: bytes.NewReader(pRand),
		}
		u, err := g.NewV7()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := u.Version(), V7; got != want {
			t.Errorf("got version %d, want %d", got, want)
		}
		if got, want := u.Variant(), VariantRFC4122; got != want {
			t.Errorf("got variant %d, want %d", got, want)
		}
		if got, want := u.String()[:15], "017f22e2-79b0-7"; got != want {
			t.Errorf("got version %q, want %q", got, want)
		}
	}
}

func makeTestNewV7Basic10000000() func(t *testing.T) {
	return func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		g := NewGen()

		for i := 0; i < 10000000; i++ {
			u, err := g.NewV7()
			if err != nil {
				t.Fatal(err)
			}
			if got, want := u.Version(), V7; got != want {
				t.Errorf("got version %d, want %d", got, want)
			}
			if got, want := u.Variant(), VariantRFC4122; got != want {
				t.Errorf("got variant %d, want %d", got, want)
			}
		}
	}
}

func makeTestNewV7DifferentAcrossCalls() func(t *testing.T) {
	return func(t *testing.T) {
		g := NewGen()

		u1, err := g.NewV7()
		if err != nil {
			t.Fatal(err)
		}
		u2, err := g.NewV7()
		if err != nil {
			t.Fatal(err)
		}
		if u1 == u2 {
			t.Errorf("generated identical UUIDs across calls: %v", u1)
		}
	}
}

func makeTestNewV7StaleEpoch() func(t *testing.T) {
	return func(t *testing.T) {
		g := &Gen{
			epochFunc: func() time.Time {
				return time.Unix(0, 0)
			},
			rand: rand.Reader,
		}
		u1, err := g.NewV7()
		if err != nil {
			t.Fatal(err)
		}
		u2, err := g.NewV7()
		if err != nil {
			t.Fatal(err)
		}
		if u1 == u2 {
			t.Errorf("generated identical UUIDs across calls: %v", u1)
		}
	}
}

func makeTestNewV7StaleEpochWithOptions() func(t *testing.T) {
	return func(t *testing.T) {
		g := NewGenWithOptions(
			WithEpochFunc(func() time.Time {
				return time.Unix(0, 0)
			}),
		)
		u1, err := g.NewV7()
		if err != nil {
			t.Fatal(err)
		}
		u2, err := g.NewV7()
		if err != nil {
			t.Fatal(err)
		}
		if u1 == u2 {
			t.Errorf("generated identical UUIDs across calls: %v", u1)
		}
	}
}

func makeTestNewV7FaultyRand() func(t *testing.T) {
	return func(t *testing.T) {
		g := &Gen{
			epochFunc: time.Now,
			rand: &faultyReader{
				readToFail: 0,
			},
		}
		u, err := g.NewV7()
		if err == nil {
			t.Errorf("got %v, nil error for clockSequence", u)
		}

		g = &Gen{
			epochFunc: time.Now,
			rand: &faultyReader{
				readToFail: 1,
			},
		}
		u, err = g.NewV7()
		if err == nil {
			t.Errorf("got %v, nil error rand_b", u)
		}
	}
}

func makeTestNewV7FaultyRandWithOptions() func(t *testing.T) {
	return func(t *testing.T) {
		g := NewGenWithOptions(
			WithRandomReader(&faultyReader{
				readToFail: 0, // fail immediately
			}),
		)
		u, err := g.NewV7()
		if err == nil {
			t.Errorf("got %v, nil error", u)
		}
	}
}

func makeTestNewV7ShortRandomRead() func(t *testing.T) {
	return func(t *testing.T) {
		g := &Gen{
			epochFunc: time.Now,
			rand:      bytes.NewReader([]byte{42}),
		}
		u, err := g.NewV7()
		if err == nil {
			t.Errorf("got %v, nil error", u)
		}
	}
}

func makeTestNewV7ShortRandomReadWithOptions() func(t *testing.T) {
	return func(t *testing.T) {
		g := NewGenWithOptions(
			WithRandomReader(bytes.NewReader([]byte{42})),
		)
		u, err := g.NewV7()
		if err == nil {
			t.Errorf("got %v, nil error", u)
		}
	}
}

func makeTestNewV7KSortable() func(t *testing.T) {
	return func(t *testing.T) {
		uuids := make([]UUID, 10)
		for i := range uuids {
			u, err := NewV7()
			testErrCheck(t, "NewV7()", "", err)

			uuids[i] = u
			time.Sleep(time.Millisecond)
		}

		for i := 1; i < len(uuids); i++ {
			p, n := uuids[i-1], uuids[i]
			isLess := p.String() < n.String()
			if !isLess {
				t.Errorf("uuids[%d] (%s) not less than uuids[%d] (%s)", i-1, p, i, n)
			}
		}
	}
}

func makeTestNewV7ClockSequence() func(t *testing.T) {
	return func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		g := NewGen()
		//always return the same TS
		g.epochFunc = func() time.Time {
			return time.UnixMilli(1645557742000)
		}
		//by being KSortable with the same timestamp, it means the sequence is Not empty, and it is monotonic
		uuids := make([]UUID, 10)
		for i := range uuids {
			u, err := g.NewV7()
			testErrCheck(t, "NewV7()", "", err)
			uuids[i] = u
		}

		for i := 1; i < len(uuids); i++ {
			p, n := uuids[i-1], uuids[i]
			isLess := p.String() < n.String()
			if !isLess {
				t.Errorf("uuids[%d] (%s) not less than uuids[%d] (%s)", i-1, p, i, n)
			}
		}
	}
}

func TestDefaultHWAddrFunc(t *testing.T) {
	tests := []struct {
		n  string
		fn func() ([]net.Interface, error)
		hw net.HardwareAddr
		e  string
	}{
		{
			n: "Error",
			fn: func() ([]net.Interface, error) {
				return nil, errors.New("controlled failure")
			},
			e: "controlled failure",
		},
		{
			n: "NoValidHWAddrReturned",
			fn: func() ([]net.Interface, error) {
				s := []net.Interface{
					{
						Index:        1,
						MTU:          1500,
						Name:         "test0",
						HardwareAddr: net.HardwareAddr{1, 2, 3, 4},
					},
					{
						Index:        2,
						MTU:          1500,
						Name:         "lo0",
						HardwareAddr: net.HardwareAddr{5, 6, 7, 8},
					},
				}

				return s, nil
			},
			e: "uuid: no HW address found",
		},
		{
			n: "ValidHWAddrReturned",
			fn: func() ([]net.Interface, error) {
				s := []net.Interface{
					{
						Index:        1,
						MTU:          1500,
						Name:         "test0",
						HardwareAddr: net.HardwareAddr{1, 2, 3, 4},
					},
					{
						Index:        2,
						MTU:          1500,
						Name:         "lo0",
						HardwareAddr: net.HardwareAddr{5, 6, 7, 8, 9, 0},
					},
				}

				return s, nil
			},
			hw: net.HardwareAddr{5, 6, 7, 8, 9, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.n, func(t *testing.T) {
			// set the netInterfaces variable (function) for the test
			// and then set it back to default in the deferred function
			netInterfaces = tt.fn
			defer func() {
				netInterfaces = net.Interfaces
			}()

			var hw net.HardwareAddr
			var err error

			hw, err = defaultHWAddrFunc()

			if len(tt.e) > 0 {
				if err == nil {
					t.Fatalf("defaultHWAddrFunc() error = <nil>, should contain %q", tt.e)
				}

				if !strings.Contains(err.Error(), tt.e) {
					t.Fatalf("defaultHWAddrFunc() error = %q, should contain %q", err.Error(), tt.e)
				}

				return
			}

			if err != nil && tt.e == "" {
				t.Fatalf("defaultHWAddrFunc() error = %q, want <nil>", err.Error())
			}

			if !bytes.Equal(hw, tt.hw) {
				t.Fatalf("hw = %#v, want %#v", hw, tt.hw)
			}
		})
	}
}

func BenchmarkGenerator(b *testing.B) {
	b.Run("NewV1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewV1()
		}
	})
	b.Run("NewV3", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewV3(NamespaceDNS, "www.example.com")
		}
	})
	b.Run("NewV4", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewV4()
		}
	})
	b.Run("NewV5", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewV5(NamespaceDNS, "www.example.com")
		}
	})
}

type faultyReader struct {
	callsNum   int
	readToFail int // Read call number to fail
}

func (r *faultyReader) Read(dest []byte) (int, error) {
	r.callsNum++
	if (r.callsNum - 1) == r.readToFail {
		return 0, fmt.Errorf("io: reader is faulty")
	}
	return rand.Read(dest)
}

// testErrCheck looks to see if errContains is a substring of err.Error(). If
// not, this calls t.Fatal(). It also calls t.Fatal() if there was an error, but
// errContains is empty. Returns true if you should continue running the test,
// or false if you should stop the test.
func testErrCheck(t *testing.T, name string, errContains string, err error) bool {
	t.Helper()

	if len(errContains) > 0 {
		if err == nil {
			t.Fatalf("%s error = <nil>, should contain %q", name, errContains)
			return false
		}

		if errStr := err.Error(); !strings.Contains(errStr, errContains) {
			t.Fatalf("%s error = %q, should contain %q", name, errStr, errContains)
			return false
		}

		return false
	}

	if err != nil && len(errContains) == 0 {
		t.Fatalf("%s unexpected error: %v", name, err)
		return false
	}

	return true
}
