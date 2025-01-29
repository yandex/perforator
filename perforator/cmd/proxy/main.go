package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/perforator/internal/buildinfo/cobrabuildinfo"
	"github.com/yandex/perforator/perforator/internal/symbolizer/proxy/server"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/maxprocs"
	"github.com/yandex/perforator/perforator/pkg/mlock"
	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/polyheapprof"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	cachePath          string
	configPath         string
	logLevel           string
	grpcPort           uint32
	httpPort           uint32
	metricsPort        uint32
	profileAllocations bool
)

var (
	proxyCmd = &cobra.Command{
		Use:   "proxy",
		Short: "Start proxy server",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx := context.Background()

			if profileAllocations {
				err := polyheapprof.StartHeapProfileRecording()
				if err != nil {
					return fmt.Errorf("failed to setup heap profiler: %w", err)
				}
			}

			level, err := log.ParseLevel(logLevel)
			if err != nil {
				return err
			}

			logger, err := xlog.TryNew(zap.NewDeployLogger(level))
			if err != nil {
				return err
			}

			err = mlock.LockExecutableMappings()
			if err == nil {
				logger.Info(ctx, "Locked self executable")
			} else {
				logger.Error(ctx, "Failed to lock self executable", log.Error(err))
			}

			conf, err := server.ParseConfig(configPath)
			if err != nil {
				return err
			}

			reg := xmetrics.NewRegistry(
				xmetrics.WithAddCollectors(xmetrics.GetCollectFuncs()...),
			)

			if cachePath != "" {
				conf.BinaryProvider.FileCache.RootPath = cachePath
			}

			serv, err := server.NewPerforatorServer(conf, logger, reg)
			if err != nil {
				return err
			}

			return serv.Run(
				ctx,
				&server.RunConfig{
					MetricsPort: metricsPort,
					HTTPPort:    httpPort,
					GRPCPort:    grpcPort,
				},
			)
		},
	}
)

func init() {
	proxyCmd.Flags().StringVar(
		&cachePath,
		"cache-path",
		"",
		"Path to symbolizer cache storing binaries and ...",
	)
	must.Must(proxyCmd.MarkFlagFilename("cache-path"))

	proxyCmd.Flags().StringVarP(
		&configPath,
		"config",
		"c",
		"",
		"Path to perforator service config",
	)
	must.Must(proxyCmd.MarkFlagFilename("config"))

	proxyCmd.Flags().StringVar(
		&logLevel,
		"log-level",
		"info",
		"Logging level - ('info') {'debug', 'info', 'warn', 'error'}",
	)

	proxyCmd.Flags().Uint32Var(
		&grpcPort,
		"grpc-port",
		80,
		"Port to start symbolizer grpc server on",
	)

	proxyCmd.Flags().Uint32Var(
		&httpPort,
		"http-port",
		81,
		"Port to start symbolizer http server on",
	)

	proxyCmd.Flags().Uint32Var(
		&metricsPort,
		"metrics-port",
		85,
		"Port to start metrics server on",
	)

	proxyCmd.Flags().BoolVar(
		&profileAllocations,
		"profile-allocations",
		false,
		"Whether to profile allocations",
	)

	cobrabuildinfo.Init(proxyCmd)
}

func main() {
	maxprocs.Adjust()
	if err := proxyCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
