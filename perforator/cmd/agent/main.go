package mainn

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
	"github.com/yandex/perforator/perforator/internal/buildinfo/cobrabuildinfo"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/maxprocs"
	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/polyheapprof"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	rootCmd = &cobra.Command{
		Use:           "agent",
		Short:         "Gather performance profiles and send them to storage",
		Long:          "Profiling agent tracing different cgroups' processes, sending profiles and binaries to storage",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return run()
		},
	}

	dumpElf          bool
	debug            bool
	configPath       string
	cgroupConfigPath string
	cgroups          []string
	pids             []int
	tids             []int
	logLevel         string
	enablePHP        bool
)

func init() {
	rootCmd.Flags().BoolVarP(&dumpElf, "dumpelf", "d", false, "dump eBPF ELF to stdout and exit")
	rootCmd.Flags().BoolVarP(&debug, "debug", "D", false, "force debug mode")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to profiler config")
	rootCmd.Flags().StringVar(&cgroupConfigPath, "cgroups", "", "path to cgroups config")
	rootCmd.Flags().StringSliceVarP(&cgroups, "cgroup", "G", nil, "name of cgroup to trace")
	rootCmd.Flags().IntSliceVarP(&pids, "pid", "p", nil, "id of process(es) to trace")
	rootCmd.Flags().IntSliceVarP(&tids, "tid", "t", nil, "id of thread(s) to trace")
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "log level (default - `info`, must be one of `debug`, `info`, `warn`, `error`)")
	rootCmd.Flags().BoolVar(&enablePHP, "enable-php", false, "[experimental feature] enable PHP profiling")

	cobrabuildinfo.Init(rootCmd)

	must.Must(rootCmd.MarkFlagFilename("config"))
	rootCmd.MarkFlagsOneRequired("dumpelf", "config")
	must.Must(rootCmd.MarkFlagFilename("cgroups"))
}

func main() {
	maxprocs.Adjust()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}

type CgroupsConfig struct {
	Cgroups []*profiler.CgroupConfig `yaml:"cgroups"`
}

func parseYaml(l log.Logger, path string, conf interface{}) error {
	if path == "" {
		l.Warn("No config file specified, using default")
		return nil
	}

	l.Info("Loading config file", log.String("path", path))
	configFile, err := os.Open(path)
	if err != nil {
		return err
	}

	yamlConfString, err := io.ReadAll(configFile)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(yamlConfString, conf)
}

func run() error {
	if dumpElf {
		reqs := unwinder.ProgramRequirements{
			Debug: debug,
			PHP:   enablePHP,
		}
		prog, err := unwinder.LoadProg(reqs)
		if err != nil {
			return fmt.Errorf("failed to load program: %w", err)
		}
		_, err = io.Copy(os.Stdout, bytes.NewReader(prog))
		return err
	}

	r := xmetrics.NewRegistry(
		xmetrics.WithAddCollectors(xmetrics.GetCollectFuncs()...),
		xmetrics.WithFormat(xmetrics.FormatText),
	)

	logLevelZap, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	l, stop, err := newLogger(logLevelZap, r)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer stop()

	c := &agent.Config{}
	err = parseYaml(l.Logger(), configPath, c)
	if err != nil {
		return err
	}
	if debug {
		c.Profiler.Debug = debug
	}

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to detect hostname: %w", err)
	}

	cgroupsConfig := &CgroupsConfig{}
	if cgroupConfigPath != "" {
		err = parseYaml(l.Logger(), cgroupConfigPath, cgroupsConfig)
		if err != nil {
			return err
		}
	}

	for _, cgroup := range cgroups {
		cgroupsConfig.Cgroups = append(cgroupsConfig.Cgroups, &profiler.CgroupConfig{
			Name: cgroup,
			Labels: map[string]string{
				"host": hostname,
			},
		})
	}

	if c.DebugModeToggler == nil {
		c.DebugModeToggler = &agent.DebugModeTogglerConfig{
			Interval:    time.Second,
			TogglerPath: "perforator.debug",
		}
	}
	agentOpts := []agent.Option{
		agent.WithDebugModeToggler(c.DebugModeToggler),
		agent.WithAgentGateway(c.AgentGateway),
	}
	if c.CPOService != nil {
		agentOpts = append(agentOpts, agent.WithCPOService(c.CPOService))
	}

	profilerOpts := []profiler.Option{}
	profilerOpts = append(profilerOpts, profiler.WithSelfTarget(map[string]string{
		"service": "perforator",
		"host":    hostname,
	}))

	for _, cgroupConfig := range cgroupsConfig.Cgroups {
		profilerOpts = append(profilerOpts, profiler.WithCgroupTarget(cgroupConfig))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, pid := range pids {
		l.Info(ctx, "Register pid", log.Int("pid", pid))
		profilerOpts = append(profilerOpts, profiler.WithProcessTarget(pid, map[string]string{
			"host": hostname,
		}))
	}

	for _, tid := range tids {
		l.Info(ctx, "Register tid", log.Int("tid", tid))
		profilerOpts = append(profilerOpts, profiler.WithThreadTarget(tid, map[string]string{
			"host": hostname,
		}))
	}

	agentOpts = append(agentOpts, agent.WithProfilerOptions(profilerOpts...))

	perforatorAgent, err := agent.NewPerforatorAgent(
		l.Logger(),
		r,
		&c.Profiler,
		agentOpts...,
	)
	if err != nil {
		return err
	}

	// Setup http puller server
	http.Handle("/metrics", r.HTTPHandler(ctx, l))
	err = polyheapprof.StartHeapProfileRecording()
	if err != nil {
		return fmt.Errorf("failed to start heap profiling")
	}

	http.HandleFunc("GET /debug/pprof/polyheap", polyheapprof.ServeCurrentHeapProfile)

	// Run pprof server
	go func() {
		err := http.ListenAndServe(":9156", nil)
		if err != nil {
			l.Error(ctx, "Failed to run http server", log.Error(err))
		}
	}()

	return perforatorAgent.Run(ctx)
}

func newLogger(level zapcore.Level, reg xmetrics.Registry) (l xlog.Logger, stop func(), err error) {
	logger, stopLogger, err := xlog.ForDaemon(xlog.DaemonConfig{}, reg)
	if err != nil {
		return nil, nil, err
	}
	return logger, stopLogger, nil
}

var prometheusMetricSanitizer = strings.NewReplacer(
	".", "_",
	"-", "_",
)

func sanitizePrometheusMetricName(name string) string {
	return prometheusMetricSanitizer.Replace(name)
}
