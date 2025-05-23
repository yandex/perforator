package clickhouse

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/yandex/perforator/perforator/pkg/certifi"
)

type Config struct {
	Replicas                    []string                `yaml:"replicas"`
	Database                    string                  `yaml:"db"`
	User                        string                  `yaml:"user"`
	PasswordEnvironmentVariable string                  `yaml:"password_env"`
	TLS                         certifi.ClientTLSConfig `yaml:"tls"`

	// TODO: all the followng fields should be replaced with
	// TLSConfig (https://github.com/yandex/perforator/blob/283248e4d7c0bd8c66c9ff28178fb635be5581ab/perforator/pkg/storage/client/client.go#L114)
	PlaintextDeprecated  bool   `yaml:"plaintext,omitempty"`
	InsecureSkipVerify   bool   `yaml:"insecure,omitempty"`
	CACertPathDeprecated string `yaml:"ca_cert_path,omitempty"`
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
}

func Connect(ctx context.Context, conf *Config) (driver.Conn, error) {
	conf.FillDefault()
	password := os.Getenv(conf.PasswordEnvironmentVariable)

	tlsConf, err := conf.TLS.BuildTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to configure TLS: %w", err)
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: conf.Replicas,
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

	return conn, nil
}
