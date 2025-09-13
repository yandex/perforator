package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/perforator/internal/agent_gateway"
	"github.com/yandex/perforator/perforator/internal/agent_gateway/storage"
	"github.com/yandex/perforator/perforator/internal/buildinfo/cobrabuildinfo"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/maxprocs"
	"github.com/yandex/perforator/perforator/pkg/mlock"
	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/pkg/xlog/logmetrics"
)

func calcProbableOutcome(probabilityPercent uint32) bool {
	return uint32(rand.UintN(100)) < probabilityPercent
}

var (
	storageConfigPath                 string
	storagePort                       uint32
	metricsPort                       uint32
	logLevel                          string
	clusterName                       string
	profileSamplingModulo             uint32
	profileSamplingModuloByEvent      map[string]int64
	maxBuildIDCacheEntries            uint64
	pushProfileTimeout                time.Duration
	writeReplicaPushBinaryProbability uint32

	storageCmd = &cobra.Command{
		Use:   "storage",
		Short: "Run storage server",
		RunE: func(*cobra.Command, []string) error {
			ctx := context.Background()

			registry := xmetrics.NewRegistry()

			level, err := log.ParseLevel(logLevel)
			if err != nil {
				return err
			}

			logBackend, err := zap.NewDeployLogger(level)
			if err != nil {
				return err
			}
			logger := xlog.New(logmetrics.NewMeteredLogger(logBackend, registry))

			err = mlock.LockExecutableMappings()
			if err != nil {
				logger.Error(ctx, "Failed to lock self executable", log.Error(err))
			}

			conf, err := agent_gateway.ParseConfig(storageConfigPath, false /* strict */)
			if err != nil {
				logger.Fatal(ctx, "Failed to parse config", log.Error(err))
			}

			if storagePort != 0 {
				conf.Port = storagePort
			}
			if metricsPort != 0 {
				conf.MetricsPort = metricsPort
			}

			// pflag does not support StringToUInt64Var, so we need to cast modulos.
			samplingModuloByEvent := make(map[string]uint64)
			for event, modulo := range profileSamplingModuloByEvent {
				samplingModuloByEvent[event] = uint64(modulo)
			}

			server, err := agent_gateway.NewServer(
				conf,
				logger,
				registry,
				storage.WithClusterName(clusterName),
				storage.WithMaxBuildIDCacheEntries(maxBuildIDCacheEntries),
				storage.WithPushProfileTimeout(pushProfileTimeout),
				storage.WithSamplingModulo(uint64(profileSamplingModulo)),
				storage.WithSamplingModuloByEvent(samplingModuloByEvent),
				storage.WithPushBinaryWriteAbility(calcProbableOutcome(uint32(writeReplicaPushBinaryProbability))),
			)
			if err != nil {
				return err
			}

			return server.Run(ctx)
		},
	}
)

func init() {
	storageCmd.Flags().StringVarP(&storageConfigPath, "config", "c", "", "Path to the config file")
	storageCmd.Flags().Uint32VarP(&storagePort, "port", "p", 0, "Port to start grpc server on")
	storageCmd.Flags().Uint32Var(&metricsPort, "metrics-port", 0, "Port to start metrics server on")
	storageCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level")
	storageCmd.Flags().Uint32Var(
		&profileSamplingModulo,
		"profile-sampling-modulo",
		1,
		"Determines how many profiles will be dropped, e.g. 1 - 0%, 2 - 50%, 10 - 90%, 100 - 99%",
	)
	storageCmd.Flags().StringToInt64Var(
		&profileSamplingModuloByEvent,
		"profile-sampling-modulo-by-event",
		make(map[string]int64),
		"Allows to override profile-sampling-modulo based on profile event type",
	)
	storageCmd.Flags().Uint32Var(
		&writeReplicaPushBinaryProbability,
		"push-binary-write-replica-probability-percent",
		15,
		"Percent probability of replica being able to push binaries into storage",
	)
	storageCmd.Flags().Uint64Var(
		&maxBuildIDCacheEntries,
		"max-build-id-cache-entries",
		14000000,
		"Build id cache max size - can reduce CPU load of storage",
	)
	storageCmd.Flags().DurationVar(
		&pushProfileTimeout,
		"push-profile-timeout",
		10*time.Second,
		"Push profile timeout",
	)
	storageCmd.Flags().StringVar(
		&clusterName,
		"cluster",
		os.Getenv("DEPLOY_NODE_DC"),
		"Name of the datacenter",
	)

	cobrabuildinfo.Init(storageCmd)

	must.Must(storageCmd.MarkFlagFilename("config"))
	must.Must(storageCmd.MarkFlagRequired("config"))
}

func main() {
	maxprocs.Adjust()
	if err := storageCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
