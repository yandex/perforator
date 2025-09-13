package clickhouse

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/cenkalti/backoff/v4"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/certifi"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type RetryConfig struct {
	MaxAttempts uint32        `yaml:"max_attempts"`
	Timeout     time.Duration `yaml:"timeout"`
}

type Config struct {
	Protocol                    string                  `yaml:"protocol"`
	Replicas                    []string                `yaml:"replicas"`
	Database                    string                  `yaml:"db"`
	User                        string                  `yaml:"user"`
	PasswordEnvironmentVariable string                  `yaml:"password_env"`
	TLS                         certifi.ClientTLSConfig `yaml:"tls"`
	ReadRetry                   RetryConfig             `yaml:"read_retry"`

	// TODO: all the followng fields should be replaced with
	// TLSConfig (https://github.com/yandex/perforator/blob/283248e4d7c0bd8c66c9ff28178fb635be5581ab/perforator/pkg/storage/client/client.go#L114)
	PlaintextDeprecated  bool   `yaml:"plaintext,omitempty"`
	InsecureSkipVerify   bool   `yaml:"insecure,omitempty"`
	CACertPathDeprecated string `yaml:"ca_cert_path,omitempty"`
}

func convertStringToProtocol(protocol string) (clickhouse.Protocol, error) {
	switch protocol {
	case clickhouse.Native.String():
		return clickhouse.Native, nil
	case clickhouse.HTTP.String():
		return clickhouse.HTTP, nil
	default:
		return 0, fmt.Errorf("invalid clickhouse protocol: %s", protocol)
	}
}

func (c *Config) FillDefault() {
	// TLS backward compatibility.
	// Previously, clickhouse client used TLS by default, so we need to enable tls if these values ​​are present.
	if c.CACertPathDeprecated != "" {
		c.TLS.Enabled = true
		c.TLS.CAFile = c.CACertPathDeprecated
	}

	if c.InsecureSkipVerify {
		c.TLS.Enabled = true
		c.TLS.InsecureSkipVerify = c.InsecureSkipVerify
	}

	if c.PlaintextDeprecated {
		c.TLS.Enabled = false
	}

	if c.Protocol == "" {
		c.Protocol = clickhouse.Native.String()
	}

	if c.ReadRetry.Timeout == 0 {
		c.ReadRetry.Timeout = 30 * time.Second
	}

	if c.ReadRetry.MaxAttempts == 0 {
		c.ReadRetry.MaxAttempts = 3
	}
}

func Connect(ctx context.Context, conf *Config) (*Connection, error) {
	conf.FillDefault()
	password := os.Getenv(conf.PasswordEnvironmentVariable)

	tlsConf, err := conf.TLS.BuildTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to configure TLS: %w", err)
	}

	protocol, err := convertStringToProtocol(conf.Protocol)
	if err != nil {
		return nil, err
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Protocol: protocol,
		Addr:     conf.Replicas,
		Auth: clickhouse.Auth{
			Database: conf.Database,
			Username: conf.User,
			Password: password,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionZSTD,
		},
		TLS:          tlsConf,
		DialTimeout:  time.Second * 10,
		MaxOpenConns: 200,
		MaxIdleConns: 300,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open clickhouse cluster connection: %w", err)
	}

	err = conn.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping clickhouse cluster: %w", err)
	}

	return newConnection(conn, conf.ReadRetry), nil
}

type Connection struct {
	driver.Conn
	readRetryConf RetryConfig
}

func newConnection(conn driver.Conn, readRetryConf RetryConfig) *Connection {
	return &Connection{
		Conn:          conn,
		readRetryConf: readRetryConf,
	}
}

func QueryWithRetries[T any](l xlog.Logger, ctx context.Context, conn *Connection, query string, scanOneRow func(driver.Rows) (T, error), args ...any) ([]T, error) {
	var result []T

	operation := func() error {
		rows, err := conn.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		result = []T{}
		for rows.Next() {
			item, err := scanOneRow(rows)
			if err != nil {
				return fmt.Errorf("failed to scan row: %w", err)
			}
			result = append(result, item)
		}

		return rows.Err()
	}

	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.MaxElapsedTime = conn.readRetryConf.Timeout

	retryBackoff := backoff.WithMaxRetries(backoffConfig, uint64(conn.readRetryConf.MaxAttempts-1))

	ctx, cancel := context.WithTimeout(ctx, conn.readRetryConf.Timeout)
	defer cancel()

	err := backoff.RetryNotify(operation, backoff.WithContext(retryBackoff, ctx), func(err error, duration time.Duration) {
		l.Warn(
			ctx,
			"SELECT failed",
			log.String("query", query),
			log.Duration("next_retry_in", duration),
			log.Error(err),
		)
	})
	return result, err
}
