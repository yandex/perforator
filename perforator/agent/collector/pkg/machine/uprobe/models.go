package uprobe

import "github.com/cilium/ebpf"

type Config struct {
	Path        string `yaml:"path"`
	Symbol      string `yaml:"symbol"`
	LocalOffset uint64 `yaml:"symbol_offset"`
	// SampleType which will be used for samples caused by this uprobe
	// If not set the default sample type "uprobes.count" will be used
	SampleType string `yaml:"sample_type"`
}

type Uprobe interface {
	// Attach attaches the uprobe to the program.
	// It can be called multiple times but uprobe must be closed before attaching again.
	Attach(prog *ebpf.Program) error

	// Close closes the uprobe bpf link.
	Close() error
}
