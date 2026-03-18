package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

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
	clusterTopIsHeavy             bool
	clusterTopDegreeOfParallelism uint

	clusterTopSchedulerGenerationInterval time.Duration
	clusterTopSchedulerProfileLag         time.Duration
	clusterTopSchedulerMaxServices        int
	clusterTopSchedulerHeavyPercent       float64
	clusterTopSchedulerMaxConflictErrors  uint32

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

			clusterTop, err := cluster_top.NewClusterTop(conf, logger, reg, storageBundle)
			if err != nil {
				return err
			}

			serviceSelector := cluster_top.NewPgServiceSelector(storageBundle.DBs.PostgresCluster)

			clusterPerfTopAggregator := cluster_top.NewClickhousePerfTopAggregator(storageBundle.ClusterTopGenerationsStorage)

			return clusterTop.Run(
				ctx,
				serviceSelector,
				clusterPerfTopAggregator,
				clusterTopIsHeavy,
				clusterTopDegreeOfParallelism,
			)
		},
	}

	clusterTopSchedulerCommand = &cobra.Command{
		Use:   "scheduler",
		Short: "Run the cluster-top generation scheduler",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

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

			schedulerConf := &scheduler.Config{
				GenerationInterval: clusterTopSchedulerGenerationInterval,
				ProfileLag:         clusterTopSchedulerProfileLag,
				MaxServices:        clusterTopSchedulerMaxServices,
				HeavyPercent:       clusterTopSchedulerHeavyPercent,
				MaxConflictErrors:  clusterTopSchedulerMaxConflictErrors,
			}
			schedulerConf.FillDefault()

			s := scheduler.NewScheduler(logger, reg, storageBundle, schedulerConf)

			return s.Run(ctx)
		},
	}
)

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
	}

	clusterTopCommand.Flags().UintVarP(
		&clusterTopDegreeOfParallelism,
		"parallelism",
		"p",
		4,
		"Degree of parallelism. Cores available is a good choice",
	)

	clusterTopCommand.Flags().BoolVar(
		&clusterTopIsHeavy,
		"heavy",
		false,
		`Whether to parallelise services processing (default), or profiles processing within a service.`,
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

	clusterTopSchedulerCommand.Flags().IntVar(
		&clusterTopSchedulerMaxServices,
		"max-services",
		10000,
		"Maximum number of services to build cluster top for",
	)

	clusterTopSchedulerCommand.Flags().Float64Var(
		&clusterTopSchedulerHeavyPercent,
		"heavy-percent",
		10.0,
		"Percentage of selected services to classify as 'heavy'",
	)

	clusterTopSchedulerCommand.Flags().Uint32Var(
		&clusterTopSchedulerMaxConflictErrors,
		"max-conflict-errors",
		3,
		"Maximum number of consecutive concurrent schedulers errors before shutting down",
	)

	clusterTopCommand.AddCommand(clusterTopSchedulerCommand)

	rootCmd.AddCommand(clusterTopCommand)
}
