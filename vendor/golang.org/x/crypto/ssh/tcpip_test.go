// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"context"
	"io"
	"net"
	"testing"
	"time"
)

func TestAutoPortListenBroken(t *testing.T) {
	broken := "SSH-2.0-OpenSSH_5.9hh11"
	works := "SSH-2.0-OpenSSH_6.1"
	if !isBrokenOpenSSHVersion(broken) {
		t.Errorf("version %q not marked as broken", broken)
	}
	if isBrokenOpenSSHVersion(works) {
		t.Errorf("version %q marked as broken", works)
	}
}

func TestClientImplementsDialContext(t *testing.T) {
	type ContextDialer interface {
		DialContext(context.Context, string, string) (net.Conn, error)
	}
	// Belt and suspenders assertion, since package net does not
	// declare a ContextDialer type.
	var _ ContextDialer = &net.Dialer{}
	var _ ContextDialer = &Client{}
}

func TestClientDialContextWithCancel(t *testing.T) {
	c := &Client{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.DialContext(ctx, "tcp", "localhost:1000")
	if err != context.Canceled {
		t.Errorf("DialContext: got nil error, expected %v", context.Canceled)
	}
}

func TestClientDialContextWithDeadline(t *testing.T) {
	c := &Client{}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()
	_, err := c.DialContext(ctx, "tcp", "localhost:1000")
	if err != context.DeadlineExceeded {
		t.Errorf("DialContext: got nil error, expected %v", context.DeadlineExceeded)
	}
}

// forwardingPair establishes a client/server connection pair over an
// in-memory pipe. The server grants all global requests and accepts all
// channels opened by the client, delivering them on the returned channel.
func forwardingPair(t *testing.T) (*Client, *ServerConn, <-chan Channel) {
	t.Helper()

	c1, c2, err := netPipe()
	if err != nil {
		t.Fatalf("netPipe: %v", err)
	}
	serverDone := make(chan struct{})
	t.Cleanup(func() { <-serverDone })
	t.Cleanup(func() {
		c1.Close()
		c2.Close()
	})

	serverConf := ServerConfig{
		NoClientAuth: true,
	}
	serverConf.AddHostKey(testSigners["rsa"])
	incoming := make(chan *ServerConn, 1)
	serverChans := make(chan Channel, 1)
	go func() {
		defer close(serverDone)
		conn, chans, reqs, err := NewServerConn(c1, &serverConf)
		incoming <- conn
		if err != nil {
			t.Errorf("NewServerConn error: %v", err)
			return
		}
		go func() {
			for r := range reqs {
				r.Reply(true, nil)
			}
		}()
		for newCh := range chans {
			ch, chanReqs, err := newCh.Accept()
			if err != nil {
				t.Errorf("Accept: %v", err)
				continue
			}
			go DiscardRequests(chanReqs)
			serverChans <- ch
		}
	}()

	conf := ClientConfig{
		HostKeyCallback: InsecureIgnoreHostKey(),
	}
	conn, chans, reqs, err := NewClientConn(c2, "", &conf)
	if err != nil {
		t.Fatalf("NewClientConn: %v", err)
	}
	client := NewClient(conn, chans, reqs)
	t.Cleanup(func() { client.Close() })

	server := <-incoming
	if server == nil {
		t.Fatal("Unable to get server")
	}
	t.Cleanup(func() { server.Close() })

	return client, server, serverChans
}

// floodStderr writes more than the default channel receive window (2 MiB)
// to w, which must be the stderr stream of an SSH channel. Stderr data is
// buffered and its window credit is only returned when the stream is
// read, so if the peer does not drain the stream the write blocks forever
// once the remote window is exhausted.
func floodStderr(t *testing.T, w io.Writer) {
	t.Helper()

	const payload = 4 * 1024 * 1024
	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 32*1024)
		remaining := payload
		for remaining > 0 {
			n := min(len(buf), remaining)
			nw, err := w.Write(buf[:n])
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
			t.Fatalf("write to stderr: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("write to stderr blocked: the stderr stream is not drained")
	}
}

// verifyMainStream checks that the main stream is still usable after
// stderr traffic by sending a payload from the server channel and
// reading it back from the client connection.
func verifyMainStream(t *testing.T, serverCh Channel, clientConn net.Conn) {
	t.Helper()

	want := []byte("main stream after stderr flood")
	if _, err := serverCh.Write(want); err != nil {
		t.Fatalf("write to main stream: %v", err)
	}
	got := make([]byte, len(want))
	if _, err := io.ReadFull(clientConn, got); err != nil {
		t.Fatalf("read from main stream: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("read %q from main stream, want %q", got, want)
	}
}

func TestClientDialDiscardsStderr(t *testing.T) {
	for _, network := range []string{"tcp", "unix"} {
		t.Run(network, func(t *testing.T) {
			client, _, serverChans := forwardingPair(t)

			addr := "127.0.0.1:8022"
			if network == "unix" {
				addr = "/tmp/forwarded.sock"
			}
			conn, err := client.Dial(network, addr)
			if err != nil {
				t.Fatalf("Dial(%q, %q): %v", network, addr, err)
			}
			defer conn.Close()
			serverCh := <-serverChans
			defer serverCh.Close()

			floodStderr(t, serverCh.Stderr())
			verifyMainStream(t, serverCh, conn)
		})
	}
}

func TestClientListenerDiscardsStderr(t *testing.T) {
	for _, network := range []string{"tcp", "unix"} {
		t.Run(network, func(t *testing.T) {
			client, server, _ := forwardingPair(t)

			var (
				ln      net.Listener
				payload []byte
				chType  string
				err     error
			)
			if network == "tcp" {
				ln, err = client.Listen("tcp", "127.0.0.1:8022")
				if err != nil {
					t.Fatalf("Listen: %v", err)
				}
				chType = "forwarded-tcpip"
				payload = Marshal(&forwardedTCPPayload{
					Addr:       "127.0.0.1",
					Port:       8022,
					OriginAddr: "127.0.0.5",
					OriginPort: 1234,
				})
			} else {
				ln, err = client.ListenUnix("/tmp/forwarded.sock")
				if err != nil {
					t.Fatalf("ListenUnix: %v", err)
				}
				chType = "forwarded-streamlocal@openssh.com"
				payload = Marshal(&forwardedStreamLocalPayload{
					SocketPath: "/tmp/forwarded.sock",
				})
			}
			defer ln.Close()

			// OpenChannel does not return until the channel is
			// accepted, so run Accept concurrently.
			accepted := make(chan net.Conn, 1)
			go func() {
				conn, err := ln.Accept()
				if err != nil {
					t.Errorf("Accept: %v", err)
					close(accepted)
					return
				}
				accepted <- conn
			}()

			serverCh, reqs, err := server.OpenChannel(chType, payload)
			if err != nil {
				t.Fatalf("OpenChannel(%q): %v", chType, err)
			}
			go DiscardRequests(reqs)
			defer serverCh.Close()

			conn := <-accepted
			if conn == nil {
				t.Fatal("no connection accepted")
			}
			defer conn.Close()

			floodStderr(t, serverCh.Stderr())
			verifyMainStream(t, serverCh, conn)
		})
	}
}
