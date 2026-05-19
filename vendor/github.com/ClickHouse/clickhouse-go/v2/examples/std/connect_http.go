package std

import (
	"database/sql"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
)

func ConnectHTTP() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, env.HttpPort)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
		Protocol: clickhouse.HTTP,
	})
	return conn.Ping()
}

func ConnectDSNHTTP() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	conn, err := sql.Open("clickhouse", fmt.Sprintf("http://%s:%d?username=%s&password=%s", env.Host, env.HttpPort, env.Username, env.Password))
	if err != nil {
		return err
	}
	return conn.Ping()
}
