package uprobe

import (
	"fmt"
	"os"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"

	"github.com/yandex/perforator/perforator/pkg/sampletype"
	"github.com/yandex/perforator/perforator/pkg/xelf"
)

const (
	defaultUprobeSampleKind  = sampletype.SampleTypeUprobe
	defaultUprobeProfileName = sampletype.SampleTypeUprobe
)

type Config struct {
	Path        string `yaml:"path"`
	Symbol      string `yaml:"symbol"`
	LocalOffset uint64 `yaml:"symbol_offset"`

	// PID of the process to attach the uprobe to.
	// If Pid equals to 0, the uprobe can be triggered by any process.
	Pid int `yaml:"pid,omitempty"`

	// SampleKind which will be used for samples caused by this uprobe
	// If not set the default sample kind "uprobe" will be used
	SampleKind string `yaml:"sample_type"`

	// OutputProfileName specifies the name of the profile to collect samples into.
	// If not set the default profile name "uprobe" will be used.
	OutputProfileName string `yaml:"output_profile_name,omitempty"`
}

func (c *Config) fillDefault() {
	if c.SampleKind == "" {
		c.SampleKind = defaultUprobeSampleKind
	}
	if c.OutputProfileName == "" {
		c.OutputProfileName = defaultUprobeProfileName
	}
}

type uprobe struct {
	link.Link
	config     Config
	binaryInfo *BinaryInfo
}

func NewUprobe(config Config) Uprobe {
	config.fillDefault()

	return &uprobe{
		config: config,
	}
}

func (u *uprobe) Close() error {
	if u.Link == nil {
		return nil
	}

	err := u.Link.Close()
	u.Link = nil
	u.binaryInfo = nil

	return err
}

func extractUprobeBinaryInfo(file *os.File, symbol string, localOffset uint64) (*BinaryInfo, error) {
	buildID, err := xelf.ReadBuildID(file)
	if err != nil {
		return nil, fmt.Errorf("failed to get build ID: %w", err)
	}

	offsets, err := xelf.GetSymbolFileOffsets(file, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbol offset: %w", err)
	}

	symbolOffset, ok := offsets[symbol]
	if !ok {
		return nil, fmt.Errorf("symbol not found: %s", symbol)
	}

	return &BinaryInfo{
		Offset:  symbolOffset + localOffset,
		BuildID: buildID,
	}, nil
}

func (u *uprobe) Info() *UprobeInfo {
	return &UprobeInfo{
		BinaryInfo: *u.binaryInfo,
		SymbolInfo: SymbolInfo{
			Name:        u.config.Symbol,
			LocalOffset: u.config.LocalOffset,
		},
		OutputInfo: OutputInfo{
			ProfileName: u.config.OutputProfileName,
			SampleKind:  u.config.SampleKind,
		},
	}
}

func (u *uprobe) Attach(prog *ebpf.Program) error {
	binary, err := os.Open(u.config.Path)
	if err != nil {
		return fmt.Errorf("failed to open binary: %w", err)
	}
	defer binary.Close()

	u.binaryInfo, err = extractUprobeBinaryInfo(binary, u.config.Symbol, u.config.LocalOffset)
	if err != nil {
		return err
	}

	// Use /proc/self/fd/* to avoid race condition with the binary path.
	tmpPath := fmt.Sprintf("/proc/self/fd/%d", binary.Fd())
	executable, err := link.OpenExecutable(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open executable %s: %w", tmpPath, err)
	}

	u.Link, err = executable.Uprobe("", prog, &link.UprobeOptions{
		Address: u.binaryInfo.Offset,
		PID:     int(u.config.Pid),
	})
	if err != nil {
		return fmt.Errorf("failed to create uprobe link: %w", err)
	}

	return nil
}
