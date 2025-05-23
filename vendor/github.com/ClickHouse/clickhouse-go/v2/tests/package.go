package tests

import (
	"os"
	"strings"
	"testing"
)

var LocalClickHouse = false
var RemoteClickHouse = false
var CloudClickHouse = false

func init() {
	if host, found := os.LookupEnv("CLICKHOUSE_HOST"); found {
		if strings.HasSuffix(host, "clickhouse.cloud") ||
			strings.HasSuffix(host, "clickhouse-staging.com") {
			CloudClickHouse = true
		} else {
			RemoteClickHouse = true
		}
	} else {
		LocalClickHouse = true
	}
}

func SkipOnCloud(t *testing.T, reasons ...string) {
	if CloudClickHouse {
		t.Skip(append(
			[]string{"Skipping test on cloud ClickHouse"},
			reasons...,
		))
	}
}
