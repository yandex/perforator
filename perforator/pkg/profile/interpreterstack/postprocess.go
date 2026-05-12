package interpreterstack

import (
	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/pkg/profile/php"
	"github.com/yandex/perforator/perforator/pkg/profile/python"
	proto "github.com/yandex/perforator/perforator/proto/perforator"
)

type options struct {
	mergePython               bool
	mergePHP                  bool
	pythonPrettificationLevel python.PrettifyLevel
}

func defaultOptions() options {
	return options{
		mergePython:               true,
		mergePHP:                  true,
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

// OptionsFromProto converts PostprocessOptions proto to a list of Option.
// By default (nil proto or nil fields) both Python and PHP merging are enabled.
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
}

// Postprocess merges interpreter (Python, PHP) stacks with native eBPF-collected stacks.
// By default both Python and PHP merging are enabled.
// Each language postprocessor detects relevant samples by examining mappings
// and skips samples that don't belong to it.
//
// Execution order:
//  1. Merge Python interpreter frames with native stack
//  2. Merge PHP interpreter frames with native stack (operates on the result of step 1)
//  3. Prettify Python stacks (if enabled) — runs last to clean up after all merges
func Postprocess(p *pprof.Profile, opts ...Option) PostProcessResults {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	var res PostProcessResults

	if o.mergePython {
		res.Python = python.Postprocess(p)
	}

	if o.mergePHP {
		php.Postprocess(p)
	}

	if o.pythonPrettificationLevel != python.PrettifyOff {
		python.PrettifyProfile(p, o.pythonPrettificationLevel)
	}

	return res
}
