package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type sshSessionOptions struct {
	username  string
	server    string
	dir       string
	identity  string
	verbose   bool
	extraOpts []string
}

func escapePath(path string) (string, error) {
	if strings.ContainsAny(path, "*\\'?+[]{}()$") {
		// TODO: actually escape?
		return "", fmt.Errorf("path %q contains invalid characters", path)
	}

	return path, nil
}

type sftpCommandUpload struct {
	src string
	dst string
}

func (c *sftpCommandUpload) serialize() (string, error) {
	s, err := escapePath(c.src)
	if err != nil {
		return "", fmt.Errorf("failed to escape source path: %w", err)
	}
	d, err := escapePath(c.dst)
	if err != nil {
		return "", fmt.Errorf("failed to escape destination path: %w", err)
	}
	return fmt.Sprintf("put '%s' '%s'", s, d), nil
}

type sftpCommandProgess struct {
}

func (c *sftpCommandProgess) serialize() (string, error) {
	return "progress", nil
}

type sftpCommandDownload struct {
	src string
	dst string
}

func (c *sftpCommandDownload) serialize() (string, error) {
	s, err := escapePath(c.src)
	if err != nil {
		return "", fmt.Errorf("failed to escape source path: %w", err)
	}
	d, err := escapePath(c.dst)
	if err != nil {
		return "", fmt.Errorf("failed to escape destination path: %w", err)
	}
	return fmt.Sprintf("get -R '%s' '%s'", s, d), nil
}

type sftpCommand struct {
	upload   *sftpCommandUpload
	download *sftpCommandDownload
	progress *sftpCommandProgess
}

func commonArgs(opts sshSessionOptions) []string {
	var args []string
	if opts.identity != "" {
		args = append(args, "-i", opts.identity)
	}
	if opts.verbose {
		args = append(args, "-v")
	}
	args = append(args, "-o", "StrictHostKeyChecking=no", "-o", "ServerAliveInterval=30")
	for _, opt := range opts.extraOpts {
		args = append(args, "-o", opt)
	}
	target := opts.server
	if opts.username != "" {
		target = fmt.Sprintf("%s@%s", opts.username, target)
	}
	if opts.dir != "" {
		target = fmt.Sprintf("%s:%s", target, opts.dir)
	}
	args = append(args, target)
	return args
}

type bufferingWriter struct {
	inner io.Writer
	limit int
	buf   []byte
}

func (bw *bufferingWriter) Write(buf []byte) (int, error) {
	n, err := bw.inner.Write(buf)
	bw.buf = append(bw.buf, buf[:n]...)
	sz := len(bw.buf)
	if sz > bw.limit {
		bw.buf = bw.buf[sz-bw.limit:]
	}
	return n, err
}

type transientSSHError struct {
	inner error
}

func (e *transientSSHError) Error() string {
	return "transient error occured: " + e.inner.Error()
}

func (e *transientSSHError) Unwrap() error {
	return e.inner
}

func runSSH(ctx context.Context, opts sshSessionOptions, commands []string) error {
	args := commonArgs(opts)
	args = append(args, commands...)
	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = bytes.NewBufferString(strings.Join(commands, "\n"))
	stderrWrapper := &bufferingWriter{
		inner: os.Stderr,
		limit: 256,
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = stderrWrapper
	execErr := cmd.Run()
	if execErr != nil && strings.Contains(string(stderrWrapper.buf), "client_loop: send disconnect: Broken pipe") {
		return &transientSSHError{
			inner: execErr,
		}
	}
	return execErr
}

func runSFTP(ctx context.Context, opts sshSessionOptions, commands []sftpCommand) error {
	var rawCommands []string
	for _, c := range commands {
		var raw string
		var err error
		switch {
		case c.upload != nil:
			raw, err = c.upload.serialize()
		case c.progress != nil:
			raw, err = c.progress.serialize()
		case c.download != nil:
			raw, err = c.download.serialize()
		default:
			err = fmt.Errorf("unknown command %+v", c)
		}
		if err != nil {
			return fmt.Errorf("failed to serialize command: %w", err)
		}
		rawCommands = append(rawCommands, raw)
	}
	sftpArgs := append([]string{"-b", "-", "-N"}, commonArgs(opts)...)
	cmd := exec.CommandContext(ctx, "sftp", sftpArgs...)
	cmd.Stdin = bytes.NewBufferString(strings.Join(rawCommands, "\n"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute commands: %w", err)
	}
	return nil
}
