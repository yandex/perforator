package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/perforator/internal/offline_processing/cluster_top"
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
	clusterTopDegreeOfParallelism *uint

	clusterTopCommand = &cobra.Command{
		Use:   "cluster-top",
		Short: "Calculate the 'perf-top' for the service",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			logLevel, err := log.ParseLevel(clusterTopLogLevelStr)
			if err != nil {
				return err
			}

			logger, err := xlog.TryNew(zap.NewDeployLogger(logLevel))
			if err != nil {
				return err
			}

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

			reg := xmetrics.NewRegistry(
				xmetrics.WithAddCollectors(xmetrics.GetCollectFuncs()...),
			)

			storageBundle, err := createStorageBundle(ctx, logger, reg, conf)
			if err != nil {
				return err
			}

			clusterTop, err := cluster_top.NewClusterTop(conf, logger, reg, storageBundle)
			if err != nil {
				return err
			}

			serviceSelector := cluster_top.NewPgServiceSelector(storageBundle.DBs.PostgresCluster)

			clusterPerfTopAggregator := cluster_top.NewClickhousePerfTopAggregator(storageBundle.DBs.ClickhouseConn)

			return clusterTop.Run(
				ctx,
				serviceSelector,
				clusterPerfTopAggregator,
				clusterTopIsHeavy,
				*clusterTopDegreeOfParallelism,
			)
		},
	}
)

func init() {
	clusterTopCommand.Flags().StringVarP(
		&clusterTopConfigPath,
		"config",
		"c",
		"",
		"Path to offline-processing config",
	)
	must.Must(clusterTopCommand.MarkFlagFilename("config"))

	clusterTopCommand.Flags().StringVar(
		&clusterTopLogLevelStr,
		"log-level",
		"info",
		"Logging level - ('info') {'debug', 'info', 'warn', 'error'}",
	)

	clusterTopDegreeOfParallelism = clusterTopCommand.Flags().UintP(
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

	rootCmd.AddCommand(clusterTopCommand)
}
