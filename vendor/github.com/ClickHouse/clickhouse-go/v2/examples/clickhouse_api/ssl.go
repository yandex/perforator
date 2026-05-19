package clickhouse_api

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func SSLVersion() error {
	env, err := GetNativeTestEnvironment()
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
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, env.SslPort)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
		TLS: t,
	})
	if err != nil {
		return err
	}
	v, err := conn.ServerVersion()
	if err != nil {
		return err
	}
	fmt.Println(v.String())
	return nil
}
