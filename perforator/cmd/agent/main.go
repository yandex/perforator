package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	"github.com/yandex/perforator/library/go/core/log"
	logzap "github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/library/go/core/log/zap/asynczap"
	"github.com/yandex/perforator/library/go/core/log/zap/encoders"
	"github.com/yandex/perforator/library/go/core/metrics/collect/policy/inflight"
	"github.com/yandex/perforator/library/go/core/metrics/prometheus"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
	"github.com/yandex/perforator/perforator/internal/buildinfo/cobrabuildinfo"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/maxprocs"
	"github.com/yandex/perforator/perforator/pkg/must"
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
	logLevel         string
)

func init() {
	rootCmd.Flags().BoolVarP(&dumpElf, "dumpelf", "d", false, "dump eBPF ELF to stdout and exit")
	rootCmd.Flags().BoolVarP(&debug, "debug", "D", false, "force debug mode")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to profiler config")
	rootCmd.Flags().StringVar(&cgroupConfigPath, "cgroups", "", "path to cgroups config")
	rootCmd.Flags().StringSliceVarP(&cgroups, "cgroup", "G", nil, "name of cgroup to trace")
	rootCmd.Flags().IntSliceVarP(&pids, "pid", "p", nil, "id of process to trace")
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "log level (default - `info`, must be one of `debug`, `info`, `warn`, `error`)")

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
		_, err := io.Copy(os.Stdout, bytes.NewReader(unwinder.LoadProg(debug)))
		return err
	}

	logLevelZap, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	l, stop, err := newLogger(logLevelZap)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer stop()

	r := prometheus.NewRegistry(prometheus.NewRegistryOpts().
		SetStreamFormat(prometheus.StreamText).
		SetNameSanitizer(sanitizePrometheusMetricName).
		AddCollectors(context.Background(),
			inflight.NewCollectorPolicy(),
			xmetrics.GetCollectFuncs()...,
		),
	)

	c := &config.Config{}
	err = parseYaml(l, configPath, c)
	if err != nil {
		return err
	}
	if debug {
		c.Debug = debug
	}

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to detect hostname: %w", err)
	}

	cgroupsConfig := &CgroupsConfig{}
	if cgroupConfigPath != "" {
		err = parseYaml(l, cgroupConfigPath, cgroupsConfig)
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

	p, err := profiler.NewProfiler(c, l, r)
	if err != nil {
		return err
	}

	err = p.TracePid(os.Getpid(), map[string]string{
		"service": "perforator",
		"host":    hostname,
	})
	if err != nil {
		return err
	}

	err = p.TraceCgroups(cgroupsConfig.Cgroups)
	if err != nil {
		return err
	}

	for _, pid := range pids {
		l.Info("Register pid", log.Int("pid", pid))
		err := p.TracePid(pid, map[string]string{
			"host": hostname,
		})
		if err != nil {
			return fmt.Errorf("failed to start pid %d tracing: %w", pid, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		tick := time.NewTicker(time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}

			if _, err := os.Stat("perforator.debug"); err == nil {
				err = p.SetDebugMode(true)
				if err != nil {
					l.Error("Failed to enable debug mode", log.Error(err))
				}
			} else {
				err = p.SetDebugMode(false)
				if err != nil {
					l.Error("Failed to disable debug mode", log.Error(err))
				}
			}
		}
	}()

	// Dump metrics to stderr every second
	go func() {
		var m runtime.MemStats
		tick := time.NewTicker(time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			_, _ = r.Stream(ctx, os.Stderr)
			_, _ = os.Stderr.WriteString("\n")

			runtime.ReadMemStats(&m)

			fmt.Fprintf(os.Stderr, "MEM: Alloc: %s, TotalAlloc: %s, Sys: %s, Mallocs: %d, Frees: %d, HeapIdle: %s, HeapInuse: %s\n",
				humanize.Bytes(m.Alloc),
				humanize.Bytes(m.TotalAlloc),
				humanize.Bytes(m.Sys),
				m.Mallocs,
				m.Frees,
				humanize.Bytes(m.HeapIdle),
				humanize.Bytes(m.HeapInuse),
			)
			fmt.Fprintf(os.Stderr, "GC: Num GC: %v, Next GC: %v, Last GC: %v, CPU fraction: %v\n",
				m.NumGC,
				humanize.Bytes(m.NextGC),
				time.Since(time.UnixMicro(int64(m.LastGC/1000))),
				m.GCCPUFraction,
			)
		}
	}()

	// Setup http puller server
	http.HandleFunc("/metrics", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		res.Header().Add("Content-Type", string(prometheus.StreamCompact))
		_, err := r.Stream(req.Context(), res)
		if err != nil {
			l.Error("Failed to stream prometheus metrics", log.Error(err))
		}
	})

	// Run pprof server
	go func() {
		err := http.ListenAndServe(":9156", nil)
		if err != nil {
			l.Error("Failed to run http server", log.Error(err))
		}
	}()

	return p.Run(ctx)
}

func newLogger(level zapcore.Level) (l log.Logger, stop func(), err error) {
	encoderconf := zap.NewProductionEncoderConfig()
	encoderconf.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	encoder, err := encoders.NewTSKVEncoder(encoderconf)
	if err != nil {
		return nil, nil, err
	}

	core := asynczap.NewCore(encoder, zapcore.Lock(os.Stdout), level, asynczap.Options{
		FlushInterval: time.Second,
	})

	return logzap.NewWithCore(core), core.Stop, nil
}

var prometheusMetricSanitizer = strings.NewReplacer(
	".", "_",
	"-", "_",
)

func sanitizePrometheusMetricName(name string) string {
	return prometheusMetricSanitizer.Replace(name)
}
