package std

import (
	"database/sql"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
)

func MultiStdHost() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9001", "127.0.0.1:9002", fmt.Sprintf("%s:%d", env.Host, env.Port)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
		ConnOpenStrategy: clickhouse.ConnOpenRoundRobin,
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

func MultiStdHostDSN() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	conn, err := sql.Open("clickhouse", fmt.Sprintf("clickhouse://127.0.0.1:9001,127.0.0.1:9002,%s:%d?username=%s&password=%s&connection_open_strategy=round_robin", env.Host, env.Port, env.Username, env.Password))
	if err != nil {
		return err
	}
	return conn.Ping()
}
