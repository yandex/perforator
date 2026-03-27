package jvmregistry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sync/errgroup"
)

// cleanupSubprocess ensures that no previously launched helper instances
// (either by earlier retries or by previous profiler processes) are alive and
// cleans up them.
func (r *Registry) cleanupSubprocess(ctx context.Context) error {
	// remove any leftover unix socket in advance, otherwise KeepAlive client
	// may wrongly connect to it
	err := os.Remove(r.helperSocketPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to cleanup leftover helper socket: %w", err)
	}

	return nil
}

func (r *Registry) runHelperSubprocess(ctx context.Context) error {
	// before starting new subprocess, let's cleanup after the old one.
	if err := r.cleanupSubprocess(ctx); err != nil {
		return fmt.Errorf("failed to cleanup subprocess: %w", err)
	}

	helperFlags := []string{
		"--socket-path",
		r.helperSocketPath,
		"--map-prefix",
		r.mapPrefix,
	}
	sp := exec.CommandContext(ctx, r.helperBinaryPath, helperFlags...)
	sp.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	sp.Stdout = os.Stdout
	sp.Stderr = os.Stderr
	err := sp.Start()
	if err != nil {
		return fmt.Errorf("failed to start command %s %v: %w", r.helperBinaryPath, helperFlags, err)
	}
	err = sp.Wait()
	if err == nil {
		err = errors.New("helper process completed prematurely")
	} else if isSignaled(err) && ctx.Err() != nil {
		// TODO: If our child was actually killed by someone else, but context
		// was canceled before we realised that, we will also return nil.
		// But maybe it's fine?
		return nil
	}
	return fmt.Errorf("command %s %v failed: %w", r.helperBinaryPath, helperFlags, err)
}

func (r *Registry) startHelper(ctx context.Context, eg *errgroup.Group) {
	eg.Go(func() error {
		err := r.runHelperSubprocess(ctx)
		if err != nil {
			return fmt.Errorf("helper subprocess execution error: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		<-ctx.Done()
		err := os.Remove(r.helperSocketPath)
		if err != nil {
			r.l.Warn(ctx, "Failed to cleanup")
		}
		return nil
	})
}
