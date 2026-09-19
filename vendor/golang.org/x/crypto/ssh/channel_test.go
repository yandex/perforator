// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"encoding/binary"
	"io"
	"math/bits"
	"testing"
	"time"
	"unsafe"
)

func TestMinPayloadSize(t *testing.T) {
	// 4 GiB (2^32). Declared as a var (not a const) so that int(bigPayload)
	// is a runtime conversion: a constant conversion would fail to compile
	// on 32-bit platforms with "constant 4294967296 overflows int". On
	// 32-bit the value truncates to 0 at runtime, but the is64Bit cases
	// that reference it are skipped by the runtime check below.
	var bigPayload int64 = 1 << 32

	tests := []struct {
		name       string
		maxPayload uint32
		dataLen    int
		want       uint32
		is64Bit    bool // Flag to run only on 64-bit architectures
	}{
		{
			name:       "Normal Case - Data fits in payload",
			maxPayload: 32768,
			dataLen:    1000,
			want:       1000,
		},
		{
			name:       "Normal Case - Data larger than payload",
			maxPayload: 32768,
			dataLen:    50000,
			want:       32768,
		},
		{
			name:       "Boundary Case - Data zero",
			maxPayload: 32768,
			dataLen:    0,
			want:       0,
		},
		{
			name:       "Overflow Case - Data is exactly 4GB (1<<32)",
			maxPayload: 32768,
			dataLen:    int(bigPayload),
			want:       32768,
			is64Bit:    true,
		},
		{
			name:       "Overflow Case - Data is 4GB + small amount",
			maxPayload: 32768,
			dataLen:    int(bigPayload + 100),
			want:       32768,
			is64Bit:    true,
		},
	}

	is64Bit := bits.UintSize == 64

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.is64Bit && !is64Bit {
				t.Skip("Skipping test requiring 64-bit int")
			}
			got := minPayloadSize(tt.maxPayload, tt.dataLen)
			if got != tt.want {
				t.Errorf("minPayloadSize(%d, %d) = %d; want %d", tt.maxPayload, tt.dataLen, got, tt.want)
			}
		})
	}
}

// TestWriteExtendedNoInfiniteLoopOnLargeWrite is an end-to-end regression
// test for the integer-overflow bug in WriteExtended. Before the fix, a
// write whose len(data) was a multiple of 2^32 caused minPayloadSize to
// return 0; WriteExtended then spun forever, reserving 0 bytes per
// iteration and never advancing the data slice.
//
// We exercise the real WriteExtended path with a slice whose declared
// length is exactly 2^32. Allocating 4 GiB is unnecessary: each iteration
// only reads up to maxRemotePayload bytes from the head of the slice, and
// the loop blocks in remoteWin.reserve() once the channel window is
// exhausted — before the slice base advances past the underlying buffer.
//
// With the fix, the loop blocks in reserve(); we detect that via
// waitWriterBlocked(), then close the window to let WriteExtended return.
// With the bug, the loop never blocks and the test times out.
//
//go:nocheckptr
func TestWriteExtendedNoInfiniteLoopOnLargeWrite(t *testing.T) {
	if bits.UintSize < 64 {
		t.Skip("test requires 64-bit int to construct a slice with len >= 2^32")
	}

	reader, writer, mux := channelPair(t)
	defer reader.Close()
	defer writer.Close()
	defer mux.Close()

	// Sized to hold the full pre-update remote window so that no iteration
	// reads past the backing buffer before reserve() blocks.
	backing := make([]byte, channelWindowSize)
	var bigLen int64 = 1 << 32
	bigSlice := unsafe.Slice(&backing[0], int(bigLen))

	done := make(chan int, 1)
	go func() {
		n, _ := writer.Write(bigSlice)
		done <- n
	}()

	blocked := make(chan struct{})
	go func() {
		writer.remoteWin.waitWriterBlocked()
		close(blocked)
	}()

	select {
	case <-blocked:
		// Good — the loop made progress and is now blocked in reserve().
		// Close the window to let WriteExtended return.
		writer.remoteWin.close()
	case <-time.After(2 * time.Second):
		t.Fatal("WriteExtended did not block in reserve within 2s — minPayloadSize likely returned 0 (integer overflow regression)")
	}

	select {
	case n := <-done:
		if n == 0 {
			t.Fatalf("WriteExtended returned n=0; expected progress")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WriteExtended did not return after closing the window")
	}
}

func TestDiscardedExtendedDataReturnsWindowCredit(t *testing.T) {
	client, _, serverChans := forwardingPair(t)

	clientCh, reqs, err := client.OpenChannel("test", nil)
	if err != nil {
		t.Fatalf("OpenChannel: %v", err)
	}
	go DiscardRequests(reqs)
	defer clientCh.Close()

	serverCh := (<-serverChans).(*channel)
	defer serverCh.Close()

	// Write more than the default channel receive window (2 MiB) of
	// extended data with a type code other than stderr. The peer
	// discards such data: if it did not return the window credit the
	// write would block forever once the window is exhausted.
	const payload = 4 * 1024 * 1024
	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 32*1024)
		remaining := payload
		for remaining > 0 {
			n := min(len(buf), remaining)
			nw, err := serverCh.WriteExtended(buf[:n], 2)
			if err != nil {
				done <- err
				return
			}
			remaining -= nw
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("write of discarded extended data: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("write of discarded extended data blocked: window credit is not returned")
	}

	// The main stream must still be usable.
	want := []byte("main stream after extended data")
	if _, err := serverCh.Write(want); err != nil {
		t.Fatalf("write to main stream: %v", err)
	}
	got := make([]byte, len(want))
	if _, err := io.ReadFull(clientCh, got); err != nil {
		t.Fatalf("read from main stream: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("read %q from main stream, want %q", got, want)
	}

	// Reading the main stream payload above orders the test after all
	// preceding packets: everything received has now been either read or
	// discarded, so the full window credit must have been accounted.
	// Anything less indicates a partial-credit leak that the liveness
	// check above would not detect.
	cc := clientCh.(*channel)
	cc.windowMu.Lock()
	window := cc.myWindow + cc.myConsumed
	cc.windowMu.Unlock()
	if window != channelWindowSize {
		t.Errorf("myWindow+myConsumed = %d, want %d", window, channelWindowSize)
	}
}

func TestDiscardedExtendedDataAfterClose(t *testing.T) {
	client, _, serverChans := forwardingPair(t)

	clientCh, reqs, err := client.OpenChannel("test", nil)
	if err != nil {
		t.Fatalf("OpenChannel: %v", err)
	}
	go DiscardRequests(reqs)
	serverCh := <-serverChans
	defer serverCh.Close()

	ch := clientCh.(*channel)
	// Close the channel locally: sentClose is now set, but the channel
	// stays in the mux chanList until the peer's close arrives, so
	// in-flight data packets are still dispatched to handleData.
	if err := ch.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Deliver extended data packets with an unknown type code, enough to
	// cross the adjustWindow send threshold of 3*maxIncomingPayload.
	// The window adjust message cannot be sent after the close and
	// adjustWindow fails with io.EOF: handleData must swallow that
	// error, since any error it returns terminates the mux read loop
	// and tears down every channel on the connection.
	const packetLen = 32 * 1024
	packet := make([]byte, 13+packetLen)
	packet[0] = msgChannelExtendedData
	binary.BigEndian.PutUint32(packet[1:], ch.localId)
	binary.BigEndian.PutUint32(packet[5:], 2)
	binary.BigEndian.PutUint32(packet[9:], packetLen)
	for range 5 {
		if err := ch.handleData(packet); err != nil {
			t.Fatalf("handleData after local close: %v", err)
		}
	}
}
