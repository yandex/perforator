// Package pgo prepares optimization profiles without a Perforator backend.
package pgo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// Format identifies the optimization profile produced by an LLVM tool.
type Format string

const (
	FormatAutoFDO Format = "autofdo"
	FormatBOLT    Format = "bolt"
)

type Tools struct {
	createLLVMProfPath string
	llvmBoltPath       string
	stderr             io.Writer
}

type Option func(*Tools)

func WithCreateLLVMProfPath(path string) Option {
	return func(t *Tools) { t.createLLVMProfPath = path }
}

func WithLLVMBoltPath(path string) Option {
	return func(t *Tools) { t.llvmBoltPath = path }
}

// WithStderr streams tool diagnostics to w. A nil writer discards diagnostics.
func WithStderr(w io.Writer) Option {
	return func(t *Tools) { t.stderr = w }
}

// NewTools defaults to PATH lookup and discarding stderr.
func NewTools(opts ...Option) Tools {
	var t Tools
	for _, opt := range opts {
		opt(&t)
	}
	return t
}

// Convert runs an installed tool; it never downloads tools or modifies the ELF.
func (t Tools) Convert(ctx context.Context, format Format, binary string, input []byte) ([]byte, error) {
	var tool string
	switch format {
	case FormatAutoFDO:
		tool = t.createLLVMProfPath
		if tool == "" {
			tool = "create_llvm_prof"
		}
	case FormatBOLT:
		tool = t.llvmBoltPath
		if tool == "" {
			tool = "llvm-bolt"
		}
	default:
		return nil, fmt.Errorf("unsupported PGO format %q: expected autofdo or bolt", format)
	}
	path, err := exec.LookPath(tool)
	if err != nil {
		return nil, fmt.Errorf("find %s: install the LLVM tool or specify its path: %w", tool, err)
	}
	output, err := os.CreateTemp("", "perforator-pgo-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(output.Name())
	if err := output.Close(); err != nil {
		return nil, err
	}
	var args []string
	if format == FormatAutoFDO {
		args = []string{
			// "profiler" selects the input format (perf, text, etc.).
			"--profiler", "text",
			// Text output is human-readable, but makes compilation much slower.
			// To inspect extbinary output, convert it with:
			// llvm-profdata merge --sample --text profile.extbinary > profile.text
			"--format", "extbinary",
			"--binary", binary,
			"--out", output.Name(),
			"--profile", "/dev/stdin",
		}
	} else {
		args = []string{
			binary,
			"--perfdata", "/dev/stdin",
			"--pa",                     // Pre-aggregated LBR profile.
			"--aggregate-only",         // Write the profile without optimizing the binary.
			"--profile-format", "yaml", // Required for stale profile matching (-infer-stale-profile).
			"-o", output.Name(),
		}
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.WaitDelay = time.Second
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stderr = t.stderr
	if err := cmd.Run(); err != nil {
		if cause := context.Cause(ctx); cause != nil {
			return nil, fmt.Errorf("%s cancelled: %w", tool, cause)
		}
		return nil, fmt.Errorf("%s failed: %w", tool, err)
	}
	data, err := os.ReadFile(output.Name())
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%s produced an empty profile", tool)
	}
	return data, nil
}
