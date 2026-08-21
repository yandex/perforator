package agent

import (
	"context"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/programstate"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/process"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

// Registry is the Python integration point for the agent profiler (cf. jvmregistry.Registry):
// BPF name/filename symbolization, optional lineno offsets, and ProcessStack for samples.
type Registry struct {
	offsets    *offsetsRegistry
	symbolizer *Symbolizer
	processor  *StackProcessor
}

type options struct {
	lineInfo bool
}

type Option func(*options)

func WithLineInfo() Option {
	return func(o *options) {
		o.lineInfo = true
	}
}

func NewRegistry(
	l xlog.Logger,
	cfg SymbolizerConfig,
	state *programstate.State,
	reg metrics.Registry,
	opts ...Option,
) (*Registry, error) {
	base, err := symbolizer.NewPythonSymbolizer(&cfg.SymbolizerConfig, state, reg)
	if err != nil {
		return nil, err
	}
	return newRegistry(l, base, cfg, reg, opts...)
}

func newRegistry(
	l xlog.Logger,
	symbols SymbolSource,
	cfg SymbolizerConfig,
	reg metrics.Registry,
	opts ...Option,
) (*Registry, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	var offsets *offsetsRegistry
	if o.lineInfo {
		offsets = newOffsetsRegistry(l)
	}
	sym, err := NewSymbolizer(symbols, offsets, cfg)
	if err != nil {
		return nil, err
	}

	return &Registry{
		offsets:    offsets,
		symbolizer: sym,
		processor:  NewStackProcessor(sym, reg),
	}, nil
}

func (r *Registry) ProcessStack(
	builder *profile.SampleBuilder,
	stack *unwinder.PythonStack,
	pid uint32,
) {
	if r == nil || r.processor == nil {
		return
	}
	r.processor.Process(builder, stack, pid)
}

// Stop releases background resources owned by the registry.
func (r *Registry) Stop() {
	if r == nil || r.symbolizer == nil {
		return
	}
	r.symbolizer.Stop()
}

func (r *Registry) OnBinaryDiscovery(ctx context.Context, binaryID uint64, buildID string, analysis *parse.BinaryAnalysis) {
	if r == nil || r.offsets == nil {
		return
	}
	r.offsets.OnBinaryDiscovery(ctx, binaryID, buildID, analysis)
}

func (r *Registry) OnProcessDiscovery(ctx context.Context, info process.ProcessInfo) {
	if r == nil || r.offsets == nil {
		return
	}
	if r.offsets.OnProcessDiscovery(ctx, info) {
		r.invalidateProcessImage(info.ProcessID())
	}
}

func (r *Registry) OnProcessRescan(ctx context.Context, info process.ProcessInfo) {
	if r == nil || r.offsets == nil {
		return
	}
	if r.offsets.OnProcessRescan(ctx, info) {
		r.invalidateProcessImage(info.ProcessID())
	}
}

func (r *Registry) OnProcessDeath(ctx context.Context, pid linux.CurrentNamespacePID) {
	if r == nil || r.offsets == nil {
		return
	}
	r.offsets.OnProcessDeath(ctx, pid)
	r.invalidateProcessImage(pid)
}

func (r *Registry) invalidateProcessImage(pid linux.CurrentNamespacePID) {
	// A process-image change invalidates both resolved tables and negative
	// linetable tombstones left by the previous image.
	r.symbolizer.InvalidatePid(uint32(pid))
}
