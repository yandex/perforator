package clickhouse_api

import (
	"crypto/tls"
	"math/rand"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
)

const TestSet string = "examples_clickhouse_api"

func GetNativeConnection(settings clickhouse.Settings, tlsConfig *tls.Config, compression *clickhouse.Compression) (driver.Conn, error) {
	return clickhouse_tests.GetConnectionTCP(TestSet, settings, tlsConfig, compression)
}

func GetHTTPConnection(sessionName string, settings clickhouse.Settings, tlsConfig *tls.Config, compression *clickhouse.Compression) (driver.Conn, error) {
	return clickhouse_tests.GetConnectionHTTP(TestSet, sessionName, settings, tlsConfig, compression)
}

func GetNativeTestEnvironment() (clickhouse_tests.ClickHouseTestEnvironment, error) {
	return clickhouse_tests.GetTestEnvironment(TestSet)
}

func GetNativeConnectionWithOptions(settings clickhouse.Settings, tlsConfig *tls.Config, compression *clickhouse.Compression) (driver.Conn, error) {
	return clickhouse_tests.GetConnectionTCP(TestSet, settings, tlsConfig, compression)
}

func CheckMinServerVersion(conn driver.Conn, major, minor, patch uint64) bool {
	return clickhouse_tests.CheckMinServerServerVersion(conn, major, minor, patch)
}

var randSeed = time.Now().UnixNano()

func ResetRandSeed() {
	rand.Seed(randSeed)
}
