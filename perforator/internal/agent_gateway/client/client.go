package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/agent_gateway/client/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/internal/agent_gateway/client/storage"
	"github.com/yandex/perforator/perforator/pkg/endpointsetresolver"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/hostname"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/interceptors/rate_limit"
	"github.com/yandex/perforator/perforator/pkg/storage/creds"
	storagetvm "github.com/yandex/perforator/perforator/pkg/storage/tvm"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	defaultCPOCallTimeout = time.Minute
)

type StorageClient = *storage.Client
type CustomProfilingOperationClient = *custom_profiling_operation.Client

type GatewayClient struct {
	StorageClient
	CustomProfilingOperationClient

	connection *grpc.ClientConn
	conf       Config
	creds      creds.DestroyablePerRPCCredentials
	logger     xlog.Logger
}

func (c *GatewayClient) Destroy() {
	_ = c.connection.Close()
	if c.creds != nil {
		c.creds.Destroy()
	}
}

func NewGatewayClient(conf *Config, l xlog.Logger) (*GatewayClient, error) {
	l = l.WithName("storage.Client")
	conf.FillDefault()

	if conf.Host == "" && conf.EndpointSet.ID == "" {
		return nil, errors.New("endpointset or host must be specified")
	}

	creds, err := getCreds(conf, l)
	if err != nil {
		return nil, err
	}

	var opts []grpc.DialOption

	serviceConfigOpts, err := setupServiceConfig(conf.Retry)
	if err != nil {
		return nil, fmt.Errorf("failed to configure service config: %w", err)
	}
	opts = append(opts, serviceConfigOpts...)

	tlsOpts, err := conf.TLS.GRPCDialOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to configure TLS: %w", err)
	}
	opts = append(opts, tlsOpts...)

	opts = append(opts,
		grpc.WithDefaultCallOptions(
			grpc.MaxSendMsgSizeCallOption{
				MaxSendMsgSize: int(conf.GRPCConfig.MaxSendMessageSize),
			},
		),
	)

	if creds != nil {
		opts = append(opts, grpc.WithPerRPCCredentials(creds))
	}

	host, err := os.Hostname()
	if err != nil {
		l.Debug(context.TODO(), "Failed to resolve hostname for request metadata", log.Error(err))
	} else if host != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(hostname.NewCredentials(host)))
	}

	rateLimitInterceptor := rate_limit.NewRateLimitInterceptor(conf.RateLimit)
	opts = append(opts, grpc.WithUnaryInterceptor(rateLimitInterceptor.UnaryInterceptor()))

	var target string
	if conf.Host != "" {
		if conf.Port != 0 {
			target = net.JoinHostPort(conf.Host, fmt.Sprint(conf.Port))
		} else {
			target = conf.Host
		}
	} else {
		endpointSetTarget, resolverOpts, err := endpointsetresolver.GetGrpcTargetAndResolverOpts(conf.EndpointSet, l)
		if err != nil {
			return nil, err
		}
		target = endpointSetTarget
		opts = append(opts, resolverOpts...)
	}

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, err
	}

	storageClient, err := storage.NewClient(&conf.StorageClient, l, conn)
	if err != nil {
		return nil, err
	}

	cpoClient := custom_profiling_operation.NewClient(conn, l)

	return &GatewayClient{
		StorageClient:                  storageClient,
		CustomProfilingOperationClient: cpoClient,
		conf:                           *conf,
		creds:                          creds,
		connection:                     conn,
		logger:                         l,
	}, nil
}

func getCreds(conf *Config, l xlog.Logger) (creds.DestroyablePerRPCCredentials, error) {
	if conf.TvmConfig != nil {
		return storagetvm.NewTVMCredentials(
			conf.TvmConfig.ServiceFromTvmID,
			conf.TvmConfig.ServiceToTvmID,
			os.Getenv(conf.TvmConfig.SecretVar),
			conf.TvmConfig.CacheDir,
			l,
		)
	}
	return nil, nil
}

// The gRPC service config only accepts time values in seconds format.
// See: https://github.com/grpc/grpc-go/blob/master/examples/features/retry/client/main.go#L42
func formatDurationForGRPC(d time.Duration) string {
	return fmt.Sprintf("%.6fs", d.Seconds())
}

// setupServiceConfig creates a gRPC dial option with the specified service config
// See:
// https://github.com/grpc/grpc/blob/master/doc/service_config.md
// https://github.com/grpc/grpc-proto/blob/master/grpc/service_config/service_config.proto
func setupServiceConfig(retryConfig RetryConfig) ([]grpc.DialOption, error) {
	retryPolicy := map[string]interface{}{
		"maxAttempts":          int(retryConfig.MaxAttempts),
		"initialBackoff":       formatDurationForGRPC(retryConfig.InitialBackoff),
		"maxBackoff":           formatDurationForGRPC(retryConfig.MaxBackoff),
		"backoffMultiplier":    retryConfig.BackoffMultiplier,
		"retryableStatusCodes": retryConfig.RetryableStatusCodes,
	}

	serviceConfig := map[string]interface{}{
		"methodConfig": []map[string]interface{}{
			{
				"name": []map[string]interface{}{
					{"service": "NPerforator.NProto.PerforatorStorage"},
				},
				"retryPolicy": retryPolicy,
			},
			{
				"name": []map[string]interface{}{
					{"service": "NPerforator.NProto.NCustomProfilingOperation.CustomProfilingOperationService"},
				},
				"retryPolicy": retryPolicy,
				"timeout":     formatDurationForGRPC(defaultCPOCallTimeout),
			},
		},
	}

	serviceConfigJSON, err := json.Marshal(serviceConfig)
	if err != nil {
		return nil, err
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithDefaultServiceConfig(string(serviceConfigJSON)))
	// Both maxAttempts fields must be specified. See: https://github.com/grpc/grpc-go/blob/v1.65.0/service_config.go#L284-L286
	opts = append(opts, grpc.WithMaxCallAttempts(retryConfig.MaxAttempts))

	return opts, nil
}
