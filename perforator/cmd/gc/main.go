package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/buildinfo/cobrabuildinfo"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/collector"
	gcconfig "github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const defaultTTL = 60 * 24 * time.Hour

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
		mux := http.NewServeMux()
		mux.Handle("/metrics", metricsHandler)
		server := &http.Server{Addr: fmt.Sprintf(":%d", metricsPort), Handler: mux}
		stop := context.AfterFunc(ctx, func() { _ = server.Close() })
		defer stop()
		l.Info(ctx, "Starting metrics server", log.UInt32("port", metricsPort))

		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	return g.Wait()
}

var (
	storageConfigPath string
	metricsPort       uint32
	logLevel          string

	profileGCConfig = gcconfig.StorageConfig{
		Type: gcconfig.Profile,
		TTL:  defaultTTL,
	}
	binaryGCConfig = gcconfig.StorageConfig{
		Type: gcconfig.Binary,
		TTL:  defaultTTL,
	}
	gsymGCConfig = gcconfig.StorageConfig{
		Type: gcconfig.GSYM,
		TTL:  defaultTTL,
	}

	iterationInterval *time.Duration
	gcLeaseName       string
	gcLeaseTTL        time.Duration

	gcCmd = &cobra.Command{
		Use:   "gc",
		Short: "Run storage garbage collector",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			r := xmetrics.NewRegistry()

			level, err := log.ParseLevel(logLevel)
			if err != nil {
				return err
			}

			logger, stopLogger, err := xlog.ForDaemon(xlog.DaemonConfig{
				Level: level,
			}, r)
			if err != nil {
				return err
			}
			defer stopLogger()

			conf, err := bundle.ParseConfig(storageConfigPath, false /* strict */)
			if err != nil {
				logger.Fatal(ctx, "Failed to parse gc config", log.Error(err))
			}

			initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			bgCtx := ctx

			bundle, err := bundle.NewStorageBundle(initCtx, bgCtx, logger, "gc", r, conf)
			if err != nil {
				logger.Fatal(ctx, "Failed to init storage bundle", log.Error(err))
			}

			gcConfig := gcconfig.Config{
				Storages:  []gcconfig.StorageConfig{},
				LeaseName: gcLeaseName,
				LeaseTTL:  gcLeaseTTL,
			}
			if conf.BinaryStorage != nil {
				gcConfig.Storages = append(gcConfig.Storages, binaryGCConfig)
				if bundle.GSYMStorage != nil {
					gcConfig.Storages = append(gcConfig.Storages, gsymGCConfig)
				}
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
	gcCmd.Flags().StringVar(&gcLeaseName, "lease-name", gcconfig.DefaultLeaseName, "Shared lease name for GC processes using the same storage config")
	gcCmd.Flags().DurationVar(&gcLeaseTTL, "lease-ttl", gcconfig.DefaultLeaseTTL, "GC lease TTL")
	gcCmd.Flags().DurationVar(
		&binaryGCConfig.TTL,
		"binary-ttl",
		defaultTTL,
		"Binary TTL, unwind table TTL is set the same",
	)
	gcCmd.Flags().DurationVar(
		&profileGCConfig.TTL,
		"profile-ttl",
		defaultTTL,
		"Profile TTL",
	)
	gcCmd.Flags().DurationVar(
		&gsymGCConfig.TTL,
		"gsym-ttl",
		defaultTTL,
		"GSYM TTL",
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
	for _, name := range []string{"profile-concurrency", "profile-shards"} {
		gcCmd.Flags().Func(name, "Deprecated; ignored", func(string) error { return nil })
		must.Must(gcCmd.Flags().MarkDeprecated(name, "ignored; GC runs one independent loop per storage"))
	}
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
	if err := gcCmd.Execute(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
