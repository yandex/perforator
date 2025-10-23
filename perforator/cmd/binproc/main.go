package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/perforator/internal/binaryprocessor"
	"github.com/yandex/perforator/perforator/internal/buildinfo/cobrabuildinfo"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/maxprocs"
	"github.com/yandex/perforator/perforator/pkg/mlock"
	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/validateconfig"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	cachePath   string
	configPath  string
	logLevel    string
	grpcPort    uint32
	metricsPort uint32
)

var (
	binprocCmd = &cobra.Command{
		Use:   "binproc",
		Short: "Start binary processor",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx := context.Background()

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

			conf, err := binaryprocessor.ParseConfig(configPath)
			if err != nil {
				return err
			}

			reg := xmetrics.NewRegistry(
				xmetrics.WithAddCollectors(xmetrics.GetCollectFuncs()...),
			)

			if cachePath != "" {
				conf.BinaryProvider.FileCache.RootPath = cachePath
			}

			serv, err := binaryprocessor.NewBinaryProcessorServer(conf, logger, reg)
			if err != nil {
				return err
			}

			return serv.Run(
				ctx,
				&binaryprocessor.RunConfig{
					GRPCPort:    grpcPort,
					MetricsPort: metricsPort,
				},
			)
		},
	}
)

func init() {
	binprocCmd.Flags().StringVar(
		&cachePath,
		"cache-path",
		"",
		"Path to symbolizer cache storing binaries and ...",
	)
	must.Must(binprocCmd.MarkFlagFilename("cache-path"))

	binprocCmd.Flags().StringVarP(
		&configPath,
		"config",
		"c",
		"",
		"Path to binary processor service config",
	)
	must.Must(binprocCmd.MarkFlagFilename("config"))

	binprocCmd.Flags().StringVar(
		&logLevel,
		"log-level",
		"info",
		"Logging level - ('info') {'debug', 'info', 'warn', 'error'}",
	)

	binprocCmd.Flags().Uint32Var(
		&grpcPort,
		"grpc-port",
		80,
		"Port to start symbolizer grpc server on",
	)

	binprocCmd.Flags().Uint32Var(
		&metricsPort,
		"metrics-port",
		85,
		"Port to start metrics server on",
	)

	binprocCmd.AddCommand(validateconfig.NewValidateConfigCmd(
		"binproc",
		validateconfig.ValidateConfigFunc(
			func(configPath string) error {
				_, err := binaryprocessor.ParseConfig(configPath)
				return err
			}),
	))

	cobrabuildinfo.Init(binprocCmd)
}

func main() {
	maxprocs.Adjust()
	if err := binprocCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
