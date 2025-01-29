package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/perforator/internal/buildinfo/cobrabuildinfo"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/collector"
	gcconfig "github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func runGC(
	ctx context.Context,
	l xlog.Logger,
	metricsHandler http.Handler,
	metricsPort uint32,
	gc *collector.GC,
	iterationInterval time.Duration,
) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return gc.Run(ctx, iterationInterval)
	})

	g.Go(func() error {
		http.Handle("/metrics", metricsHandler)
		l.Info(ctx, "Starting metrics server", log.UInt32("port", metricsPort))

		return http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), nil)
	})

	return g.Wait()
}

var (
	storageConfigPath string
	metricsPort       uint32
	logLevel          string

	profileGCConfig = gcconfig.StorageConfig{
		Type: gcconfig.Profile,
		TTL: gcconfig.TTLConfig{
			TTL: 1440 * time.Hour,
		},
		Concurrency: &gcconfig.ConcurrencyConfig{
			Shards:      1,
			Concurrency: 1,
		},
	}
	binaryGCConfig = gcconfig.StorageConfig{
		Type: gcconfig.Binary,
		TTL: gcconfig.TTLConfig{
			TTL: 1440 * time.Hour,
		},
		Concurrency: &gcconfig.ConcurrencyConfig{
			Shards:      1,
			Concurrency: 1,
		},
	}

	iterationInterval *time.Duration

	gcCmd = &cobra.Command{
		Use:   "gc",
		Short: "Run storage garbage collector",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()

			level, err := log.ParseLevel(logLevel)
			if err != nil {
				return err
			}

			logger, err := xlog.TryNew(zap.NewDeployLogger(level))
			if err != nil {
				return err
			}

			if profileGCConfig.Concurrency.Concurrency <= 0 {
				return fmt.Errorf("%d profile concurrency must be positive", profileGCConfig.Concurrency.Concurrency)
			}

			if profileGCConfig.Concurrency.Shards <= 0 {
				return fmt.Errorf("%d profile shards must be positive", profileGCConfig.Concurrency.Shards)
			}

			r := xmetrics.NewRegistry()

			conf, err := bundle.ParseConfig(storageConfigPath, false /* strict */)
			if err != nil {
				logger.Fatal(ctx, "Failed to parse gc config", log.Error(err))
			}

			initCtx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
			defer cancel()
			bundle, err := bundle.NewStorageBundle(initCtx, logger, r, conf)
			if err != nil {
				logger.Fatal(ctx, "Failed to init storage bundle", log.Error(err))
			}

			gcConfig := gcconfig.Config{
				Storages: []gcconfig.StorageConfig{},
			}
			if conf.BinaryStorage != nil {
				gcConfig.Storages = append(gcConfig.Storages, binaryGCConfig)
			}
			if conf.ProfileStorage != nil {
				gcConfig.Storages = append(gcConfig.Storages, profileGCConfig)
			}

			gc, err := collector.NewGC(
				logger,
				r,
				gcConfig,
				bundle,
			)
			if err != nil {
				return err
			}

			return runGC(
				ctx,
				logger,
				r.HTTPHandler(ctx, logger),
				metricsPort,
				gc,
				*iterationInterval,
			)
		},
	}
)

func init() {
	gcCmd.Flags().DurationVar(
		&binaryGCConfig.TTL.TTL,
		"binary-ttl",
		time.Hour*1440,
		"Binary TTL, unwind table TTL is set the same",
	)
	gcCmd.Flags().DurationVar(
		&profileGCConfig.TTL.TTL,
		"profile-ttl",
		time.Hour*1440,
		"Profile TTL",
	)

	gcCmd.Flags().StringVarP(
		&storageConfigPath,
		"config",
		"c",
		"",
		"Path to storage config",
	)
	gcCmd.Flags().Uint32Var(&metricsPort, "metrics-port", 85, "Port to export metrics on")
	iterationInterval = gcCmd.Flags().DurationP(
		"interval",
		"i",
		time.Minute,
		"Interval between gc iterations",
	)

	gcCmd.Flags().Uint32Var(&profileGCConfig.DeletePageSize, "delete-page-size", 500, "How many objects will be deleted in one try")
	gcCmd.Flags().Uint32Var(
		&profileGCConfig.Concurrency.Concurrency,
		"profile-concurrency",
		1,
		"Level of concurrency for profile GC (must be not greater than profile shards)",
	)
	gcCmd.Flags().Uint32Var(
		&profileGCConfig.Concurrency.Shards,
		"profile-shards",
		1,
		"Number of shards for concurrenct garbage collection (must be 2^x)",
	)
	gcCmd.Flags().StringVar(
		&logLevel,
		"log-level",
		"info",
		"Log level",
	)

	cobrabuildinfo.Init(gcCmd)

	must.Must(gcCmd.MarkFlagFilename("config"))
	must.Must(gcCmd.MarkFlagRequired("config"))
}

func main() {
	if err := gcCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
