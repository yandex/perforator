package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/tracing"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/symbolizer/pkg/client"
)

////////////////////////////////////////////////////////////////////////////////

type ClientConfig = client.Config

type Config struct {
	LogLevel string
	Timeout  time.Duration
	Client   *ClientConfig
}

func (c *Config) fillDefault() {
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.Timeout == time.Duration(0) {
		c.Timeout = time.Second * 30
	}
}

////////////////////////////////////////////////////////////////////////////////

type App struct {
	logger   xlog.Logger
	client   *client.Client
	shutdown func()
	context  context.Context
	cancel   func()
}

func New(config *Config) (*App, error) {
	config.fillDefault()

	var err error

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	level, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log level: %w", err)
	}

	logger, err := NewLogger(level)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		if err != nil {
			logger.Error(ctx, "Failed to initialize CLI", log.Error(err))
		}
	}()

	var rpcClient *client.Client
	if config.Client != nil {
		if !config.Client.Insecure && config.Client.Token == "" {
			token, err := findToken(ctx, logger)
			if err != nil {
				return nil, err
			}
			if token != "" {
				config.Client.Token = token
				logger.Debug(ctx, "Found OAuth token", log.Int("len", len(token)))
			}
		} else if config.Client.Insecure {
			logger.Warn(ctx, "Running in insecure mode, disabling TLS & OAuth")
		} else {
			logger.Debug(ctx, "Using provided OAuth token")
		}

		rpcClient, err = client.NewClient(config.Client, logger.WithName("client"))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize perforator client: %w", err)
		}
	}

	stop, _, err := tracing.Initialize(
		context.Background(),
		logger.WithContext(ctx).WithName("tracing"),
		tracing.NewNopExporter(),
		"perforator",
		"cli",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}
	shutdown := func() {
		_ = stop(ctx)
	}

	return &App{logger, rpcClient, shutdown, ctx, cancel}, nil
}

////////////////////////////////////////////////////////////////////////////////

func (a *App) Shutdown() {
	a.cancel()
	a.shutdown()
}

func (a *App) Client() *client.Client {
	return a.client
}

func (a *App) Logger() xlog.Logger {
	return a.logger
}

func (a *App) ContextLogger() log.Logger {
	return a.logger.WithContext(a.context)
}

func (a *App) Context() context.Context {
	return a.context
}

////////////////////////////////////////////////////////////////////////////////
