package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"go.opentelemetry.io/otel"

	"github.com/yandex/perforator/perforator/internal/symbolizer/autofdo"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	ErrNoCreateLLVMProfBinary = errors.New("create_llvm_prof binary path is not set")
	ErrNoLLVMBoltBinary       = errors.New("llvm-bolt binary path is not set")
)

type LLVMTools struct {
	l xlog.Logger

	createLLVMProfBinaryPath string
	llvmBoltBinaryPath       string
	binaryProvider           binaryprovider.BinaryProvider
}

func (t *LLVMTools) fetchBinary(ctx context.Context, buildID string) (binaryprovider.FileHandle, uint64, error) {
	binary, err := t.binaryProvider.Acquire(ctx, buildID)
	if err != nil {
		return nil, 0, err
	}
	if err = binary.WaitStored(ctx); err != nil {
		return nil, 0, err
	}

	executableBytesCount, err := autofdo.GetBinaryExecutableBytes(binary.Path())
	if err != nil {
		executableBytesCount = 0
	}

	return binary, executableBytesCount, nil
}

func NewLLVMTools(l xlog.Logger, pgoConfig *PGOConfig, binaryProvider binaryprovider.BinaryProvider) LLVMTools {
	createLLVMProfBinaryPath := ""
	llvmBoltBinaryPath := ""

	if pgoConfig != nil {
		createLLVMProfBinaryPath = pgoConfig.CreateLLVMProfBinaryPath
		llvmBoltBinaryPath = pgoConfig.LlvmBoltBinaryPath
	}

	return LLVMTools{
		l:                        l,
		createLLVMProfBinaryPath: createLLVMProfBinaryPath,
		llvmBoltBinaryPath:       llvmBoltBinaryPath,
		binaryProvider:           binaryProvider,
	}
}

type LLVMPGOProfile struct {
	profileBytes         []byte
	executableBytesCount uint64
}

func (t *LLVMTools) CreateAutofdoProfile(ctx context.Context, input []byte, buildID string) (*LLVMPGOProfile, error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "LLVMTools.CreateAutofdoProfile")
	defer span.End()

	if t.createLLVMProfBinaryPath == "" {
		return nil, ErrNoCreateLLVMProfBinary
	}

	binary, executableBytesCount, err := t.fetchBinary(ctx, buildID)
	if err != nil {
		return nil, err
	}
	defer binary.Close()

	output, err := os.CreateTemp("", "*.spgo.extbinary")
	if err != nil {
		return nil, err
	}
	defer os.Remove(output.Name())

	cmd := exec.Command(t.createLLVMProfBinaryPath,
		// "profiler" is the type of input data (perf, text, etc.)
		"--profiler", "text",
		// although "text" format is human readable, is slows down compilation way to much.
		// one could always convert it to "text" via `llvm-profdata merge --sample --text profile.extbinary > profile.text`
		"--format", "extbinary",
		"--binary", binary.Path(),
		"--out", output.Name(),
		"--profile", "/dev/stdin",
	)
	profileBytes, err := runCommand(cmd, input, output)
	if err != nil {
		return nil, err
	}

	return &LLVMPGOProfile{
		profileBytes:         profileBytes,
		executableBytesCount: executableBytesCount,
	}, nil
}

func (t *LLVMTools) CreateBoltProfile(ctx context.Context, input []byte, buildID string) (*LLVMPGOProfile, error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "LLVMTools.CreateBoltProfile")
	defer span.End()

	if t.llvmBoltBinaryPath == "" {
		return nil, ErrNoLLVMBoltBinary
	}

	binary, executableBytesCount, err := t.fetchBinary(ctx, buildID)
	if err != nil {
		return nil, err
	}
	defer binary.Close()

	output, err := os.CreateTemp("", "*.bolt.yaml")
	if err != nil {
		return nil, err
	}
	defer os.Remove(output.Name())

	cmd := exec.Command(t.llvmBoltBinaryPath,
		binary.Path(),
		"--perfdata", "/dev/stdin",
		"--pa",                     // pre-aggregated LBR profile
		"--aggregate-only",         // just write the profile, don't optimize the binary
		"--profile-format", "yaml", // yaml-format is required for stale profile matching (-infer-stale-profile flag)
		"-o", output.Name(),
	)
	profileBytes, err := runCommand(cmd, input, output)
	if err != nil {
		return nil, err
	}

	return &LLVMPGOProfile{
		profileBytes:         profileBytes,
		executableBytesCount: executableBytesCount,
	}, nil
}

func runCommand(cmd *exec.Cmd, input []byte, output *os.File) ([]byte, error) {
	readSide := bytes.NewBuffer(input)
	cmd.Stdin = readSide

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start %s: %w", cmd.Path, err)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to run %s: %w", cmd.Path, err)
	}

	profile, err := os.ReadFile(output.Name())
	if err != nil {
		return nil, err
	}

	return profile, nil
}
