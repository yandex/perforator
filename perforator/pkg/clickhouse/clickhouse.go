package clickhouse

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Config struct {
	Replicas                    []string `yaml:"replicas"`
	Database                    string   `yaml:"db"`
	User                        string   `yaml:"user"`
	PasswordEnvironmentVariable string   `yaml:"password_env"`
	InsecureSkipVerify          bool     `yaml:"insecure,omitempty"`
	CACertPath                  string   `yaml:"ca_cert_path,omitempty"`
}

func Connect(ctx context.Context, conf *Config) (driver.Conn, error) {
	password := os.Getenv(conf.PasswordEnvironmentVariable)

	tlsConf := &tls.Config{
		InsecureSkipVerify: conf.InsecureSkipVerify,
	}

	if !conf.InsecureSkipVerify && conf.CACertPath != "" {
		cert, err := os.ReadFile(conf.CACertPath)
		if err != nil {
			return nil, err
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(cert) {
			return nil, fmt.Errorf("failed to add server CA's certificate")
		}

		tlsConf.RootCAs = certPool
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
