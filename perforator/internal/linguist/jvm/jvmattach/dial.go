package jvmattach

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/pidfd"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type Dialer struct {
	Logger xlog.Logger
}

// Target contains information about target JVM.
type Target struct {
	// ProcessFD is the pidfd referring to the target process.
	ProcessFD *pidfd.FD
	// Pid in the pid namespace of this process.
	PID linux.ProcessID
	// Pid in the pid namespace of the target process.
	NamespacedPID linux.ProcessID
	// Target process cwd in the mount namespace and chroot of this process.
	// Defaults to a fallback value which relies on /proc.
	CWD string
	// Target process chroot in the mount namespace and chroot of this process.
	// Defaults to a fallback value which relies on /proc.
	Chroot string
}

func (t *Target) fillDefaults() {
	if t.CWD == "" {
		t.CWD = fmt.Sprintf("/proc/%d/cwd", t.PID)
	}
	if t.Chroot == "" {
		t.Chroot = fmt.Sprintf("/proc/%d/root", t.PID)
	}
}

func (d *Dialer) Dial(ctx context.Context, target Target) (*VirtualMachineConn, error) {
	target.fillDefaults()
	conn, err := d.dialImpl(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %d: %w", target.PID, err)
	}
	return conn, nil
}

// https://github.com/openjdk/jdk/blob/7c944ee6f4dda4f1626721d63ac6bc6d1b40d33b/src/jdk.attach/linux/classes/sun/tools/attach/VirtualMachineImpl.java#L91
const sleepDelayStep = 100 * time.Millisecond

func (d *Dialer) dialImpl(ctx context.Context, target Target) (*VirtualMachineConn, error) {
	d.Logger.Debug(ctx, "Checking if socket already exists")
	conn, err := d.tryConnect(ctx, target.Chroot, target.NamespacedPID)
	if err != nil {
		return nil, fmt.Errorf("initial socket check failed: %w", err)
	}
	if conn != nil {
		d.Logger.Debug(ctx, "Socket already exists")
		return conn, nil
	}
	err = d.sendAttachRequest(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("failed to send attach request: %w", err)
	}
	var sleepDelay time.Duration
	for i := 0; ; i++ {
		d.Logger.Info(ctx, "Attempting to connect to JVM")
		conn, err := d.tryConnect(ctx, target.Chroot, target.NamespacedPID)
		if err != nil {
			return nil, fmt.Errorf("got fatal error while attempting to connect to JVM: %w", err)
		}
		if conn != nil {
			return conn, nil
		}

		d.Logger.Info(ctx, "Failed to connect to JVM, sleeping", log.Error(err))
		sleepDelay += sleepDelayStep
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("canceled while dialing JVM: %w", context.Cause(ctx))
		case <-time.After(sleepDelay):
		}
	}
}

func (d *Dialer) sendAttachRequest(ctx context.Context, target Target) error {
	attachFilePath := path.Join(target.CWD, fmt.Sprintf(".attach_pid%d", target.NamespacedPID))
	d.Logger.Debug(ctx, "Creating attach file", log.String("path", attachFilePath))
	err := os.WriteFile(attachFilePath, []byte{}, 0644)
	if errors.Is(err, os.ErrExist) {
		d.Logger.Debug(ctx, "Attach file already exists, skipping")
	} else if err != nil {
		return fmt.Errorf("failed to create attach file %w", err)
	}
	defer func() {
		err := os.Remove(attachFilePath)
		if err != nil {
			d.Logger.Warn(ctx, "Failed to cleanup attach file", log.Error(err))
		}
	}()

	d.Logger.Info(ctx, "Sending SIGQUIT signal to JVM")
	err = target.ProcessFD.SendSignal(syscall.SIGQUIT)
	if err != nil {
		return fmt.Errorf("failed to send SIGQUIT: %w", err)
	}
	return nil
}

func (d *Dialer) tryConnect(ctx context.Context, chroot string, nspid uint32) (*VirtualMachineConn, error) {
	d.Logger.Debug(ctx, "Trying to connect")
	path := getFilePath(chroot, ".java_pid", nspid)
	raw := net.Dialer{}
	conn, err := raw.DialContext(ctx, "unix", path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			d.Logger.Debug(ctx, "Socket does not exist", log.Error(err))
			return nil, nil
		}
		d.Logger.Warn(ctx, "Connection failed", log.Error(err))
		return nil, err
	}
	err = conn.Close()
	if err != nil {
		d.Logger.Warn(ctx, "Dial cleanup failed: unable to close test connection", log.Error(err))
	}
	return &VirtualMachineConn{path: path}, nil
}

func getFilePath(chroot string, filename string, nspid uint32) string {
	return fmt.Sprintf("%s/tmp/%s%d", chroot, filename, nspid)
}
