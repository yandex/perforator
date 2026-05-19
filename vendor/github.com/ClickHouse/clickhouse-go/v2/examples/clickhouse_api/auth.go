package clickhouse_api

import (
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
)

func Auth() error {
	env, err := GetNativeTestEnvironment()
	if err != nil {
		return err
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, env.Port)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
	})
	if err != nil {
		return err
	}
	v, err := conn.ServerVersion()
	fmt.Println(v)
	if err != nil {
		return err
	}
	return nil
}
