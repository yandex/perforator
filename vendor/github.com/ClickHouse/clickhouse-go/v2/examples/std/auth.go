package std

import (
	"database/sql"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
)

func ConnectAuth() error {
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

func ConnectDSNAuth() error {
	env, err := GetStdTestEnvironment()
	conn, err := sql.Open("clickhouse", fmt.Sprintf("http://%s:%d?username=%s&password=%s", env.Host, env.HttpPort, env.Username, env.Password))
	if err != nil {
		return err
	}
	if err = conn.Ping(); err != nil {
		return err
	}
	conn, err = sql.Open("clickhouse", fmt.Sprintf("http://%s:%s@%s:%d", env.Username, env.Password, env.Host, env.HttpPort))
	if err != nil {
		return err
	}
	return conn.Ping()
}
