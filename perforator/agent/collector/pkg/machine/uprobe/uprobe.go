package uprobe

import (
	"fmt"
	"os"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

type uprobe struct {
	link.Link
	config Config
	opts   *options
	key    Key
	reg    *Registry
}

func (u *uprobe) Close() error {
	if u.Link == nil {
		return nil
	}

	err := u.Link.Close()
	u.Link = nil
	u.reg.removeResolveInfo(u.key)

	return err
}

func (u *uprobe) Attach(prog *ebpf.Program) error {
	binary, err := os.Open(u.config.Path)
	if err != nil {
		return fmt.Errorf("failed to open binary: %w", err)
	}
	defer binary.Close()

	offset, err := extractOffset(binary, u.config.Symbol, u.config.LocalOffset)
	if err != nil {
		return err
	}

	u.key, err = extractKey(binary, offset)
	if err != nil {
		return fmt.Errorf("failed to extract uprobe key: %w", err)
	}

	// Use /proc/self/fd/* to avoid race condition with the binary path.
	tmpPath := fmt.Sprintf("/proc/self/fd/%d", binary.Fd())
	executable, err := link.OpenExecutable(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open executable %s: %w", tmpPath, err)
	}

	u.Link, err = executable.Uprobe("", prog, &link.UprobeOptions{
		Address: offset,
		PID:     int(u.opts.pid),
	})
	if err != nil {
		return fmt.Errorf("failed to create uprobe link: %w", err)
	}

	u.reg.addResolveInfo(u.key, u.config.Symbol, u.config.LocalOffset, u.config.SampleType)

	return nil
}
