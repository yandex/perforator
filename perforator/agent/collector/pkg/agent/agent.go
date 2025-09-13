package agent

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
)

type agentOptions struct {
	debugModeTogglerConfig *DebugModeTogglerConfig
	profilerOpts           []profiler.Option
}

type Option func(*agentOptions)

func WithDebugModeToggler(config *DebugModeTogglerConfig) Option {
	return func(o *agentOptions) {
		o.debugModeTogglerConfig = config
	}
}

func WithProfilerOptions(opts ...profiler.Option) Option {
	return func(o *agentOptions) {
		o.profilerOpts = append(o.profilerOpts, opts...)
	}
}

type PerforatorAgent struct {
	l        log.Logger
	profiler *profiler.Profiler
	// TODO: forbid usage of this interface by adding deploy system field
	//   which will manipulate with profiler targets inside PerforatorAgent
	targetManipulator
	debugModeToggler *debugModeTogglerWatcher
	options          *agentOptions
	// TODO: add CustomProfilingOperationWorker
}

type targetManipulator interface {
	TraceSelf(labels map[string]string) error
	TraceCgroups(configs []*profiler.CgroupConfig) error
}

func NewPerforatorAgent(
	l log.Logger,
	r metrics.Registry,
	profilerConfig *config.Config,
	agentOpts ...Option,
) (*PerforatorAgent, error) {
	options := &agentOptions{}
	for _, opt := range agentOpts {
		opt(options)
	}

	profiler, err := profiler.NewProfiler(profilerConfig, l, r, options.profilerOpts...)
	if err != nil {
		return nil, err
	}

	agent := &PerforatorAgent{
		l:                 l,
		profiler:          profiler,
		targetManipulator: profiler,
		options:           options,
	}

	if options.debugModeTogglerConfig != nil {
		agent.debugModeToggler = newDebugModeTogglerWatcher(l, options.debugModeTogglerConfig, profiler)
	}

	return agent, nil
}

func (a *PerforatorAgent) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	if a.debugModeToggler != nil {
		g.Go(func() error {
			a.l.Info("Starting debug mode toggle watcher", log.String("path", a.debugModeToggler.conf.TogglerPath))
			err := a.debugModeToggler.run(ctx)
			a.l.Error("Exiting debug mode toggle watcher", log.Error(err))
			return err
		})
	}

	g.Go(func() error {
		return a.profiler.Run(ctx)
	})

	return g.Wait()
}
