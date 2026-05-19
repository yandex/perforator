package clickhouse_api

import (
	"fmt"
	"math/rand"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func MultiHostVersion() error {
	return multiHostVersion(nil)
}

func MultiHostRoundRobinVersion() error {
	connOpenStrategy := clickhouse.ConnOpenRoundRobin
	return multiHostVersion(&connOpenStrategy)
}

func MultiHostRandomVersion() error {
	rand.Seed(85206178671753424)
	defer ResetRandSeed()
	connOpenStrategy := clickhouse.ConnOpenRandom
	return multiHostVersion(&connOpenStrategy)
}

func multiHostVersion(connOpenStrategy *clickhouse.ConnOpenStrategy) error {
	env, err := GetNativeTestEnvironment()
	if err != nil {
		return err
	}
	options := clickhouse.Options{
		Addr: []string{"127.0.0.1:9001", "127.0.0.1:9002", fmt.Sprintf("%s:%d", env.Host, env.Port)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
	}
	if connOpenStrategy != nil {
		options.ConnOpenStrategy = *connOpenStrategy
	}
	conn, err := clickhouse.Open(&options)
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
