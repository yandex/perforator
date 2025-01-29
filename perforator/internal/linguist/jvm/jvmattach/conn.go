package jvmattach

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
)

// VirtualMachineConn is a connection to a JVM.
// see https://github.com/openjdk/jdk/blob/7c944ee6f4dda4f1626721d63ac6bc6d1b40d33b/src/jdk.attach/linux/classes/sun/tools/attach/VirtualMachineImpl.java
type VirtualMachineConn struct {
	path string
}

func (c *VirtualMachineConn) Execute(ctx context.Context, cmdAndArgs [4]string) (string, error) {
	stream, err := c.ExecuteStream(ctx, cmdAndArgs)
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}
	defer stream.Close()
	response, err := io.ReadAll(stream)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	return string(response), nil
}

func (c *VirtualMachineConn) ExecuteStream(ctx context.Context, cmdAndArgs [4]string) (io.ReadCloser, error) {
	raw := net.Dialer{}
	conn, err := raw.DialContext(ctx, "unix", c.path)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to socket %q: %w", c.path, err)
	}
	var errs []error

	enc := &jvmEncoder{wr: conn}
	enc.writeCommand(cmdAndArgs)
	if enc.error() != nil {
		errs = append(errs, fmt.Errorf("writing command failed: %w", enc.error()))
	}
	dec := &jvmDecoder{rd: conn}
	status, ok := dec.readInt()
	if status != 0 || !ok {
		response, _ := dec.readString()
		errs = append(errs, fmt.Errorf("jvm returned error %d: %s", status, response))
	}
	if dec.error() != nil {
		errs = append(errs, fmt.Errorf("reading response failed: %w", dec.error()))
	}
	if len(errs) > 0 {
		err = conn.Close()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to close socket: %w", err))
		}
		return nil, fmt.Errorf("jvm command failed: %w", errors.Join(errs...))
	}
	return conn, nil
}

func (c *VirtualMachineConn) Close() error {
	return nil
}
