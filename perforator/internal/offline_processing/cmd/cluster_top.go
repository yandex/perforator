package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/offline_processing/cluster_top"
	"github.com/yandex/perforator/perforator/internal/offline_processing/cluster_top/scheduler"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/mlock"
	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func createStorageBundle(
	ctx context.Context,
	l xlog.Logger,
	reg xmetrics.Registry,
	conf *cluster_top.Config,
) (*bundle.StorageBundle, error) {
	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// TODO: this context should be tied to e.g. Run() duration.
	bgCtx := context.TODO()

	storageBundle, err := bundle.NewStorageBundle(initCtx, bgCtx, l, "cluster-top", reg, &conf.Storage)
	if err != nil {
		return nil, err
	}
	l.Info(ctx, "Initialized storage bundle")

	return storageBundle, nil
}

var (
	clusterTopConfigPath          string
	clusterTopLogLevelStr         string
	clusterTopDegreeOfParallelism uint
	clusterTopMetricsPort         uint32

	clusterTopSchedulerGenerationInterval time.Duration
	clusterTopSchedulerProfileLag         time.Duration
	clusterTopSchedulerMaxConflictErrors  uint32
	clusterTopSchedulerBucketCount        uint16

	clusterTopCommand = &cobra.Command{
		Use:   "cluster-top",
		Short: "Calculate the 'perf-top' for the service",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			reg := xmetrics.NewRegistry(
				xmetrics.WithAddCollectors(xmetrics.GetCollectFuncs()...),
			)

			logLevel, err := log.ParseLevel(clusterTopLogLevelStr)
			if err != nil {
				return err
			}

			logger, stopLogger, err := xlog.ForDaemon(xlog.DaemonConfig{
				Level: logLevel,
			}, reg)
			if err != nil {
				return err
			}
			defer stopLogger()

			err = mlock.LockExecutableMappings()
			if err == nil {
				logger.Info(ctx, "Locked self executable")
			} else {
				logger.Error(ctx, "Failed to lock self executable", log.Error(err))
			}

			conf, err := cluster_top.ParseConfig(clusterTopConfigPath)
			if err != nil {
				return err
			}

			storageBundle, err := createStorageBundle(ctx, logger, reg, conf)
			if err != nil {
				return err
			}

			clusterTop, err := cluster_top.NewClusterTop(
				conf,
				logger,
				reg,
				storageBundle,
			)
			if err != nil {
				return err
			}

			if len(conf.Worker.SkippedServices) > 0 {
				logger.Info(ctx, "Cluster top worker service skip list enabled",
					log.Int("count", len(conf.Worker.SkippedServices)),
					log.Strings("services", conf.Worker.SkippedServices),
				)
			}

			jobSelector := cluster_top.NewPgJobSelector(storageBundle.DBs.PostgresCluster)

			clusterPerfTopAggregator := cluster_top.NewClickhousePerfTopAggregator(storageBundle.ClusterTopGenerationsStorage)

			g, ctx := errgroup.WithContext(ctx)

			g.Go(func() error {
				return runMetricsServer(ctx, logger, reg, clusterTopMetricsPort)
			})

			g.Go(func() error {
				return clusterTop.Run(
					ctx,
					jobSelector,
					clusterPerfTopAggregator,
					clusterTopDegreeOfParallelism,
				)
			})

			return g.Wait()
		},
	}

	clusterTopSchedulerCommand = &cobra.Command{
		Use:   "scheduler",
		Short: "Run the cluster-top generation scheduler",
		RunE: func(cmd *cobra.Command, args []string) error {
			schedulerConf := &scheduler.Config{
				GenerationInterval: clusterTopSchedulerGenerationInterval,
				ProfileLag:         clusterTopSchedulerProfileLag,
				MaxConflictErrors:  clusterTopSchedulerMaxConflictErrors,
				BucketCount:        clusterTopSchedulerBucketCount,
			}

			ctx := cmd.Context()

			logLevel, err := log.ParseLevel(clusterTopLogLevelStr)
			if err != nil {
				return err
			}

			reg := xmetrics.NewRegistry(
				xmetrics.WithAddCollectors(xmetrics.GetCollectFuncs()...),
			)

			logger, stopLogger, err := xlog.ForDaemon(xlog.DaemonConfig{Level: logLevel}, reg)
			if err != nil {
				return err
			}
			defer stopLogger()

			conf, err := cluster_top.ParseConfig(clusterTopConfigPath)
			if err != nil {
				return err
			}

			storageBundle, err := createStorageBundle(ctx, logger, reg, conf)
			if err != nil {
				return err
			}

			s := scheduler.NewScheduler(logger, reg, storageBundle, schedulerConf)

			g, ctx := errgroup.WithContext(ctx)

			g.Go(func() error {
				return runMetricsServer(ctx, logger, reg, clusterTopMetricsPort)
			})

			g.Go(func() error {
				return s.Run(ctx)
			})

			return g.Wait()
		},
	}
)

func runMetricsServer(
	ctx context.Context,
	logger xlog.Logger,
	reg xmetrics.Registry,
	port uint32,
) error {
	if port == 0 {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", reg.HTTPHandler(ctx, logger))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	logger.Info(ctx, "Starting metrics server", log.UInt32("port", port))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func init() {
	for _, cmd := range []*cobra.Command{clusterTopCommand, clusterTopSchedulerCommand} {
		cmd.Flags().StringVarP(
			&clusterTopConfigPath,
			"config",
			"c",
			"",
			"Path to offline-processing config",
		)
		must.Must(cmd.MarkFlagFilename("config"))

		cmd.Flags().StringVar(
			&clusterTopLogLevelStr,
			"log-level",
			"info",
			"Logging level - ('info') {'debug', 'info', 'warn', 'error'}",
		)

		cmd.Flags().Uint32Var(
			&clusterTopMetricsPort,
			"metrics-port",
			0,
			"Port to export metrics on (0 disables metrics server)",
		)
	}

	clusterTopCommand.Flags().UintVarP(
		&clusterTopDegreeOfParallelism,
		"parallelism",
		"p",
		4,
		"Degree of parallelism for job profile processing (number of CPU cores is a good default)",
	)

	clusterTopSchedulerCommand.Flags().DurationVar(
		&clusterTopSchedulerGenerationInterval,
		"generation-interval",
		24*time.Hour,
		"Time between generations",
	)

	clusterTopSchedulerCommand.Flags().DurationVar(
		&clusterTopSchedulerProfileLag,
		"profile-lag",
		10*time.Minute,
		"Safety buffer to allow all profiles to arrive in storage",
	)

	clusterTopSchedulerCommand.Flags().Uint32Var(
		&clusterTopSchedulerMaxConflictErrors,
		"max-conflict-errors",
		3,
		"Maximum number of consecutive concurrent schedulers errors before shutting down",
	)

	clusterTopSchedulerCommand.Flags().Uint16Var(
		&clusterTopSchedulerBucketCount,
		"partition-bucket-count",
		scheduler.DefaultBucketCount,
		"Partition bucket count frozen for each new generation",
	)
	clusterTopCommand.AddCommand(clusterTopSchedulerCommand)

	rootCmd.AddCommand(clusterTopCommand)
}
