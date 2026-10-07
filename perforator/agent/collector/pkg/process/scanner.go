package process

import (
	"context"
	"os"
	"strconv"

	"github.com/yandex/perforator/perforator/pkg/linux"
)

type ProcessScanner interface {
	// Scan must report live PIDs independently of profiling targets.
	Scan(ctx context.Context, discoverer func(context.Context, linux.CurrentNamespacePID)) error
}

////////////////////////////////////////////////////////////////////////////////

type ProcFSScanner struct{}

func (p *ProcFSScanner) Scan(ctx context.Context, discoverer func(context.Context, linux.CurrentNamespacePID)) (err error) {
	procDir, err := os.Open("/proc")
	if err != nil {
		return
	}
	defer procDir.Close()

	entries, err := procDir.ReadDir(0 /* read all dir entries */)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			pid, err := strconv.ParseUint(entry.Name(), 10, 32)
			if err != nil { // not a pid directory
				continue
			}
			discoverer(ctx, linux.CurrentNamespacePID(pid))
		}
	}
	return
}

////////////////////////////////////////////////////////////////////////////////

// ProcessFilter controls discovery of new registrations. Rejecting a PID does
// not end an existing registration's lifetime.
type ProcessFilter func(pid linux.CurrentNamespacePID) bool

////////////////////////////////////////////////////////////////////////////////

// Compile-time inheritance check.
var _ ProcessScanner = &ProcFSScanner{}
