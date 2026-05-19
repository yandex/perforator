package std

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"os"
	"path"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func ConnectSSL() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	t := &tls.Config{}
	caCert, err := os.ReadFile(path.Join(cwd, "../../tests/resources/CAroot.crt"))
	if err != nil {
		return err
	}
	caCertPool := x509.NewCertPool()
	successful := caCertPool.AppendCertsFromPEM(caCert)
	if !successful {
		return err
	}
	t.RootCAs = caCertPool

	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, env.SslPort)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
		TLS: t,
	})
	return conn.Ping()
}

func ConnectDSNSSL() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	conn, err := sql.Open("clickhouse", fmt.Sprintf("https://%s:%d?secure=true&skip_verify=true&username=%s&password=%s", env.Host, env.HttpsPort, env.Username, env.Password))
	if err != nil {
		return err
	}
	return conn.Ping()
}
