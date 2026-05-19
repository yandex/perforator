package std

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func Connect() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, env.Port)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
	})
	return conn.Ping()
}

func ConnectDSN() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	conn, err := sql.Open("clickhouse", fmt.Sprintf("clickhouse://%s:%d?username=%s&password=%s", env.Host, env.Port, env.Username, env.Password))
	if err != nil {
		return err
	}
	return conn.Ping()
}

func ConnectUsingHTTPProxy() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return fmt.Errorf("failed to get test environment: %w", err)
	}

	proxyURL, err := url.Parse("http://proxy.example.com:3128")
	if err != nil {
		return fmt.Errorf("failed to parse proxy URL: %w", err)
	}

	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, env.Port)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
		HTTPProxyURL: proxyURL,
	})
	return conn.Ping()
}

func ConnectUsingHTTPProxyDSN() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return fmt.Errorf("failed to get test environment: %w", err)
	}

	urlEncodedProxyURL := url.QueryEscape("http://proxy.example.com:3128")

	conn, err := sql.Open("clickhouse", fmt.Sprintf("clickhouse://%s:%d?username=%s&password=%s&http_proxy=%s", env.Host, env.Port, env.Username, env.Password, urlEncodedProxyURL))
	if err != nil {
		return err
	}
	return conn.Ping()
}
