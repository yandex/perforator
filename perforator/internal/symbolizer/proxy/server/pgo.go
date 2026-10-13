package server

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"

	"github.com/yandex/perforator/perforator/internal/pgo"
	"github.com/yandex/perforator/perforator/internal/symbolizer/autofdo"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	ErrNoCreateLLVMProfBinary = errors.New("create_llvm_prof binary path is not set")
	ErrNoLLVMBoltBinary       = errors.New("llvm-bolt binary path is not set")
)

type PGOGenerator struct {
	l xlog.Logger

	createLLVMProfBinaryPath string
	llvmBoltBinaryPath       string
	binaryProvider           binaryprovider.BinaryProvider
	tools                    pgo.Tools
}

func (t *PGOGenerator) fetchBinary(ctx context.Context, buildID string) (binaryprovider.FileHandle, uint64, error) {
	binary, err := t.binaryProvider.Acquire(ctx, buildID)
	if err != nil {
		return nil, 0, err
	}

	executableBytesCount, err := autofdo.GetBinaryExecutableBytes(binary.Path())
	if err != nil {
		executableBytesCount = 0
	}

	return binary, executableBytesCount, nil
}

func NewPGOGenerator(l xlog.Logger, pgoConfig *PGOConfig, binaryProvider binaryprovider.BinaryProvider) PGOGenerator {
	createLLVMProfBinaryPath := ""
	llvmBoltBinaryPath := ""

	if pgoConfig != nil {
		createLLVMProfBinaryPath = pgoConfig.CreateLLVMProfBinaryPath
		llvmBoltBinaryPath = pgoConfig.LlvmBoltBinaryPath
	}

	return PGOGenerator{
		l:                        l,
		createLLVMProfBinaryPath: createLLVMProfBinaryPath,
		llvmBoltBinaryPath:       llvmBoltBinaryPath,
		binaryProvider:           binaryProvider,
		tools: pgo.NewTools(
			pgo.WithCreateLLVMProfPath(createLLVMProfBinaryPath),
			pgo.WithLLVMBoltPath(llvmBoltBinaryPath),
		),
	}
}

type LLVMPGOProfile struct {
	profileBytes         []byte
	executableBytesCount uint64
}

func (t *PGOGenerator) CreateAutofdoProfile(ctx context.Context, input []byte, buildID string) (*LLVMPGOProfile, error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PGOGenerator.CreateAutofdoProfile")
	defer span.End()

	if t.createLLVMProfBinaryPath == "" {
		return nil, ErrNoCreateLLVMProfBinary
	}

	binary, executableBytesCount, err := t.fetchBinary(ctx, buildID)
	if err != nil {
		return nil, err
	}
	defer binary.Close()

	profileBytes, err := t.tools.Convert(ctx, pgo.FormatAutoFDO, binary.Path(), input)
	if err != nil {
		return nil, err
	}

	return &LLVMPGOProfile{
		profileBytes:         profileBytes,
		executableBytesCount: executableBytesCount,
	}, nil
}

func (t *PGOGenerator) CreateBoltProfile(ctx context.Context, input []byte, buildID string) (*LLVMPGOProfile, error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PGOGenerator.CreateBoltProfile")
	defer span.End()

	if t.llvmBoltBinaryPath == "" {
		return nil, ErrNoLLVMBoltBinary
	}

	binary, executableBytesCount, err := t.fetchBinary(ctx, buildID)
	if err != nil {
		return nil, err
	}
	defer binary.Close()

	profileBytes, err := t.tools.Convert(ctx, pgo.FormatBOLT, binary.Path(), input)
	if err != nil {
		return nil, err
	}

	return &LLVMPGOProfile{
		profileBytes:         profileBytes,
		executableBytesCount: executableBytesCount,
	}, nil
}
