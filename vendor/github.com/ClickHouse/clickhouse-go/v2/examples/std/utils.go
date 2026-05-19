package std

import (
	"crypto/tls"
	"database/sql"
	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
	clickhouse_tests_std "github.com/ClickHouse/clickhouse-go/v2/tests/std"
)

const TestSet string = "examples_std_api"

func GetStdDSNConnection(protocol clickhouse.Protocol, secure bool, compress string) (*sql.DB, error) {
	return clickhouse_tests_std.GetDSNConnection(TestSet, protocol, secure, nil)
}

func GetStdOpenDBConnection(protocol clickhouse.Protocol, settings clickhouse.Settings, tlsConfig *tls.Config, compression *clickhouse.Compression) (*sql.DB, error) {
	return clickhouse_tests_std.GetOpenDBConnection(TestSet, protocol, settings, tlsConfig, compression)
}

func GetStdTestEnvironment() (clickhouse_tests.ClickHouseTestEnvironment, error) {
	return clickhouse_tests.GetTestEnvironment(TestSet)
}

func CheckMinServerVersion(conn *sql.DB, major, minor, patch uint64) bool {
	return clickhouse_tests_std.CheckMinServerVersion(conn, major, minor, patch)
}
