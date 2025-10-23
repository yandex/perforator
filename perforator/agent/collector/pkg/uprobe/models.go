package uprobe

import "github.com/cilium/ebpf"

type Uprobe interface {
	// Info returns the uprobe info.
	// This must only be called on attached uprobe.
	Info() *UprobeInfo

	// Attach attaches the uprobe to the program.
	// It can be called multiple times but uprobe must be closed before attaching again.
	Attach(prog *ebpf.Program) error

	// Close detaches the uprobe from the program and releases the resources.
	Close() error
}
