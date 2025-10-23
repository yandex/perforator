package agent

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	agent_gateway_client "github.com/yandex/perforator/perforator/internal/agent_gateway/client"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type agentOptions struct {
	debugModeTogglerConfig *DebugModeTogglerConfig
	profilerOpts           []profiler.Option
	agentGatewayConfig     *agent_gateway_client.Config
	cpoServiceConfig       *custom_profiling_operation.ServiceConfig
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

func WithAgentGateway(config *agent_gateway_client.Config) Option {
	return func(o *agentOptions) {
		o.agentGatewayConfig = config
	}
}

func WithCPOService(config *custom_profiling_operation.ServiceConfig) Option {
	return func(o *agentOptions) {
		o.cpoServiceConfig = config
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

	agentGatewayClient *agent_gateway_client.GatewayClient
	cpoService         *custom_profiling_operation.Service
}

type targetManipulator interface {
	TraceSelf(labels map[string]string) (profiler.Closer, error)
	TraceCgroups(configs []*profiler.CgroupConfig) error
}

func NewPerforatorAgent(
	l log.Logger,
	r metrics.Registry,
	profilerConfig *config.Config,
	opts ...Option,
) (*PerforatorAgent, error) {
	options := &agentOptions{}
	for _, opt := range opts {
		opt(options)
	}

	var err error
	agent := &PerforatorAgent{
		l: l,
	}

	xLogger := xlog.New(l)

	clientConfig := options.agentGatewayConfig
	if clientConfig == nil {
		clientConfig = profilerConfig.StorageClientConfigDeprecated
	}
	if clientConfig != nil {
		agentGatewayClient, err := agent_gateway_client.NewGatewayClient(clientConfig, xLogger)
		if err != nil {
			return nil, err
		}
		agent.agentGatewayClient = agentGatewayClient
	}

	if agent.agentGatewayClient != nil {
		remoteStorage := client.NewRemoteStorage(xLogger, r, agent.agentGatewayClient.StorageClient)
		options.profilerOpts = append(options.profilerOpts, profiler.WithStorage(remoteStorage))
	}

	agent.profiler, err = profiler.NewProfiler(profilerConfig, l, r, options.profilerOpts...)
	if err != nil {
		return nil, err
	}
	agent.targetManipulator = agent.profiler

	if options.debugModeTogglerConfig != nil {
		agent.debugModeToggler = newDebugModeTogglerWatcher(l, options.debugModeTogglerConfig, agent.profiler)
	}

	if options.cpoServiceConfig != nil {
		registry := custom_profiling_operation.NewOperationExecutionRegistry(xLogger, r, agent.profiler, agent.agentGatewayClient.CustomProfilingOperationClient)
		handler := custom_profiling_operation.NewCPOHandler(xLogger, registry)
		agent.cpoService, err = custom_profiling_operation.NewService(xLogger, r, options.cpoServiceConfig, agent.agentGatewayClient.CustomProfilingOperationClient, handler)
		if err != nil {
			return nil, err
		}
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

	if a.cpoService != nil {
		g.Go(func() error {
			a.l.Info("Starting custom profiling operation service")
			err := a.cpoService.Run(ctx)
			a.l.Error("Exiting custom profiling operation service", log.Error(err))
			return err
		})
	}

	g.Go(func() error {
		return a.profiler.Run(ctx)
	})

	return g.Wait()
}
