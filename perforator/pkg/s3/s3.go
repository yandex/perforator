package s3

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/certifi"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	defaultRegion       = "us-east-1"
	defaultAccessKeyEnv = "S3_ACCESS_KEY"
	defaultSecretKeyEnv = "S3_SECRET_KEY"
	defaultMaxRetries   = 5
)

// dynamicHTTPProxyKind specifies how proxy configuration should be obtained.
type dynamicHTTPProxyKind = string

const (
	// expect plaintext proxy hostname.
	dynamicHTTPProxyKindHostname dynamicHTTPProxyKind = "hostname"
)

type DynamicHTTPProxyConfig struct {
	Kind dynamicHTTPProxyKind `yaml:"kind"`
	// URL that should be polled to get http proxy configuration
	ConfigurationEndpoint string `yaml:"configuration_endpoint"`
	// Override port. By default port returned by the configuration endpoint is used.
	OverridePort *uint16 `yaml:"override_port,omitempty"`
	// Override scheme. By default https is used.
	OverrideScheme string        `yaml:"override_scheme,omitempty"`
	UpdateInterval time.Duration `yaml:"update_interval"`
	// MaxErrorUpdates limits number of consecutive configuration refreshes caused by transport errors.
	// It is used to avoid overloading configuration service.
	MaxErrorUpdates int32 `yaml:"max_error_updates"`
}

type Config struct {
	Endpoint string `yaml:"endpoint"`

	SecretKeyPath string `yaml:"secret_key_path"`
	AccessKeyPath string `yaml:"access_key_path"`
	SecretKeyEnv  string `yaml:"secret_key_env"`
	AccessKeyEnv  string `yaml:"access_key_env"`

	Region         string `yaml:"region"`
	ForcePathStyle *bool  `yaml:"force_path_style"`

	TLS certifi.ClientTLSConfig `yaml:"tls"`

	InsecureSkipVerify   bool   `yaml:"insecure,omitempty"`
	CACertPathDeprecated string `yaml:"ca_cert_path,omitempty"`

	DynamicHTTPProxy *DynamicHTTPProxyConfig `yaml:"dynamic_http_proxy,omitempty"`
}

func (c *Config) fillDefault() {
	if c.Region == "" {
		c.Region = defaultRegion
	}
	if c.AccessKeyEnv == "" && c.AccessKeyPath == "" {
		c.AccessKeyEnv = defaultAccessKeyEnv
	}
	if c.SecretKeyEnv == "" && c.SecretKeyPath == "" {
		c.SecretKeyEnv = defaultSecretKeyEnv
	}

	// TLS backward compatibility.
	// Previously, s3 client used TLS by default, so we need to enable tls if these values ​​are present.
	if c.CACertPathDeprecated != "" {
		c.TLS.Enabled = true
		c.TLS.CAFile = c.CACertPathDeprecated
	}

	if c.InsecureSkipVerify {
		c.TLS.Enabled = true
		c.TLS.InsecureSkipVerify = c.InsecureSkipVerify
	}
}

func loadKey(path, env string) (string, error) {
	if path != "" {
		value, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read secret key from %s: %w", path, err)
		}

		return string(value), nil
	}

	if env != "" {
		value, ok := os.LookupEnv(env)
		if !ok {
			return "", fmt.Errorf("environment variable %s is not set", env)
		}

		return value, nil
	}

	return "", fmt.Errorf("no key path or environment variable provided")
}

func addRetryObserver(s3Client *s3.S3, registry metrics.Registry) {
	s3Client.Handlers.AfterRetry.PushBack(func(r *request.Request) {
		statusCode := 0
		if r.HTTPResponse != nil {
			statusCode = r.HTTPResponse.StatusCode
		}

		operation := ""
		if r.Operation != nil {
			operation = r.Operation.Name
		}

		registry.WithTags(map[string]string{
			"status_code": strconv.Itoa(statusCode),
			"operation":   operation,
		}).Counter("s3.client.retries").Inc()
	})
}

// NewClient creates a new S3 client. `refreshCtx` should be valid while client is in use.
func NewClient(ctx context.Context, refreshCtx context.Context, logger xlog.Logger, c *Config, reg metrics.Registry) (*s3.S3, error) {
	c.fillDefault()

	secretKey, err := loadKey(c.SecretKeyPath, c.SecretKeyEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret key: %w", err)
	}

	accessKey, err := loadKey(c.AccessKeyPath, c.AccessKeyEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to read access key: %w", err)
	}

	config := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, "")).
		WithEndpoint(c.Endpoint).
		WithRegion(c.Region).
		WithMaxRetries(defaultMaxRetries)

	if c.ForcePathStyle != nil {
		config = config.WithS3ForcePathStyle(*c.ForcePathStyle)
	}

	tlsConfig, err := c.TLS.BuildTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to configure TLS: %w", err)
	}

	httpTransport := http.Transport{
		TLSClientConfig: tlsConfig,
	}
	var proxy *dynamicProxy
	if c.DynamicHTTPProxy != nil {
		proxy, err = newDynamicProxy(ctx, refreshCtx, logger, reg.WithPrefix("s3.client.dynamic_http_proxy"), c.DynamicHTTPProxy)
		if err != nil {
			return nil, fmt.Errorf("failed to setup dynamic HTTP proxy: %w", err)
		}
		httpTransport.Proxy = func(r *http.Request) (*url.URL, error) {
			return proxy.proxy(), nil
		}
	}

	httpClient := &http.Client{
		Transport: &httpTransport,
	}
	config = config.WithHTTPClient(httpClient).
		WithDisableSSL(!c.TLS.Enabled) // We must explicitly pass this flag, otherwise, s3 client will use TLS by default, even when the HTTP client was created with a nil tls.Config.

	session, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	s3Client := s3.New(session)
	addRetryObserver(s3Client, reg)

	if c.DynamicHTTPProxy != nil {
		s3Client.Handlers.CompleteAttempt.PushBack(func(r *request.Request) {
			if r.HTTPResponse.StatusCode >= 500 {
				proxy.onError()
				return
			}
			if r.Error == nil {
				return
			}
			awsErr, ok := r.Error.(awserr.Error)
			if !ok {
				return
			}

			var netErr net.Error
			if errors.As(awsErr.OrigErr(), &netErr) {
				proxy.onError()
			}
		})
	}

	return s3Client, nil
}
