package certifi_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/jackc/pgx/v4"
	pgxstd "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/yandex/perforator/library/go/certifi"
)

func ExampleNewCertPool_http() {
	tlsConfig := &tls.Config{}
	certPool, err := certifi.NewCertPool()
	if err != nil {
		panic(fmt.Sprintf("failed to create cert pool: %v\n", err))
	} else {
		tlsConfig.RootCAs = certPool
	}

	httpClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	resp, err := httpClient.Get("https://tools.sec.yandex-team.ru/")
	if err != nil {
		panic(fmt.Sprintf("failed to call sectools: %v\n", err))
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	fmt.Printf("successful, status code: %d\n", resp.StatusCode)
}

func ExampleNewCertPool_pgxstd() {
	config, err := pgx.ParseConfig("host=foo.db.yandex.net port=6432 dbname=bar sslmode=verify-full user=baz")
	if err != nil {
		panic(fmt.Sprintf("failed to create db: %v\n", err))
	}

	caCertPool, err := certifi.NewCertPoolInternal()
	if err != nil {
		panic(fmt.Sprintf("failed to get InternalCA: %v\n", err))
	}

	config.Password = os.Getenv("PG_PASSWORD")
	config.TLSConfig.RootCAs = caCertPool

	driverDB := pgxstd.OpenDB(*config)
	pg := sqlx.NewDb(driverDB, "pgx")
	err = pg.Ping()
	if err != nil {
		panic(err)
	}
}
