package interpreterstack

import (
	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/pkg/profile/lua"
	"github.com/yandex/perforator/perforator/pkg/profile/php"
	"github.com/yandex/perforator/perforator/pkg/profile/python"
	proto "github.com/yandex/perforator/perforator/proto/perforator"
)

type options struct {
	mergePython               bool
	mergePHP                  bool
	mergeLua                  bool
	pythonPrettificationLevel python.PrettifyLevel
}

func defaultOptions() options {
	return options{
		mergePython:               true,
		mergePHP:                  true,
		mergeLua:                  true,
		pythonPrettificationLevel: python.PrettifyOff,
	}
}

// Option configures the behavior of Postprocess.
type Option func(*options)

// WithPythonPrettifyLevel sets the Python stack prettification level.
func WithPythonPrettifyLevel(level python.PrettifyLevel) Option {
	return func(o *options) {
		o.pythonPrettificationLevel = level
	}
}

// WithoutPython disables Python stack merging.
func WithoutPython() Option {
	return func(o *options) {
		o.mergePython = false
	}
}

// WithoutPHP disables PHP stack merging.
func WithoutPHP() Option {
	return func(o *options) {
		o.mergePHP = false
	}
}

// WithoutLua disables stitching while still removing Lua VM transport metadata.
func WithoutLua() Option {
	return func(o *options) { o.mergeLua = false }
}

// OptionsFromProto converts PostprocessOptions proto to a list of Option.
// By default (nil proto or nil fields) interpreter merging is enabled.
func OptionsFromProto(pp *proto.PostprocessOptions) []Option {
	if pp == nil {
		return nil
	}

	var opts []Option
	if pp.MergePythonAndNativeStacks != nil && !*pp.MergePythonAndNativeStacks {
		opts = append(opts, WithoutPython())
	}
	if pp.MergePHPAndNativeStacks != nil && !*pp.MergePHPAndNativeStacks {
		opts = append(opts, WithoutPHP())
	}
	if pp.MergeLuaAndNativeStacks != nil && !*pp.MergeLuaAndNativeStacks {
		opts = append(opts, WithoutLua())
	}

	if pp.PrettifyPythonStacksLevel != nil {
		switch *pp.PrettifyPythonStacksLevel {
		case proto.PythonStackPrettifyLevel_PYTHON_STACK_PRETTIFY_DEFAULT:
			opts = append(opts, WithPythonPrettifyLevel(python.PrettifyMixed))
		case proto.PythonStackPrettifyLevel_PYTHON_STACK_PRETTIFY_STRICT:
			opts = append(opts, WithPythonPrettifyLevel(python.PrettifyPythonOnly))
		default:
			// PYTHON_STACK_PRETTIFY_OFF or unknown: no prettification
		}
	}

	return opts
}

// PostProcessResults aggregates results from all interpreter stack merging passes.
type PostProcessResults struct {
	Python python.PostProcessResults
	Lua    lua.PostProcessResults
}

// Postprocess merges interpreter stacks with native eBPF-collected stacks.
// By default interpreter merging is enabled.
// Each language postprocessor detects relevant samples by examining mappings
// and skips samples that don't belong to it.
//
// Execution order:
//  1. Merge Python interpreter frames with native stack
//  2. Strip PHP frames while on-agent PHP unwinding is disabled
//  3. Prettify Python stacks (if enabled)
//  4. Consume Lua metadata and merge Lua interpreter frames (if enabled)
func Postprocess(p *pprof.Profile, opts ...Option) PostProcessResults {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	var res PostProcessResults
	if o.mergePython {
		res.Python = python.Postprocess(p)
	}

	// TODO: on-agent PHP unwinding is broken, so unconditionally strip the
	// synthetic [php] frames instead of merging them — this keeps PHP out of
	// profiles and flamegraphs. Restore php.Postprocess(p) (gated on o.mergePHP)
	// once the agent-side fix ships.
	php.Strip(p)

	if o.pythonPrettificationLevel != python.PrettifyOff {
		python.PrettifyProfile(p, o.pythonPrettificationLevel)
	}

	res.Lua = lua.Postprocess(p, o.mergeLua)

	return res
}
