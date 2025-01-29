package main

import (
	"context"
	"flag"
	"fmt"
	standardLog "log"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/perforator/internal/symbolizer/quality_monitoring/internal/config"
	"github.com/yandex/perforator/perforator/internal/symbolizer/quality_monitoring/internal/service"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
)

func main() {
	configPath := flag.String("config", "", "Path to monitoring service config")
	logLevel := flag.String("log-level", "info", "Logging level - ('info') {'debug', 'info', 'warn', 'error'}")
	metricsPort := flag.Uint("metrics-port", 85, "Port on which the metrics server is running")

	flag.Parse()

	logger, err := setupLogger(*logLevel)
	if err != nil {
		standardLog.Fatalf("can't create logger: %s", err)
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		standardLog.Fatalf("can't load config: %s", err)
	}

	reg := xmetrics.NewRegistry()

	serv, err := service.NewMonitoringService(cfg, logger, reg)
	if err != nil {
		standardLog.Fatalf("can't create monitoring server: %s", err)
	}

	err = serv.Run(
		context.Background(),
		logger,
		&service.RunConfig{
			MetricsPort: *metricsPort,
		})
	if err != nil {
		logger.Error("service is stoping with error", log.Error(err))
	}
}

func setupLogger(logLevel string) (log.Logger, error) {
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		return nil, err
	}

	logger, err := zap.NewDeployLogger(level)
	if err != nil {
		return nil, fmt.Errorf("can't create logger: %w", err)
	}

	return logger, nil
}
