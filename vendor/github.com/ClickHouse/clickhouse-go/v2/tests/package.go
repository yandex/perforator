package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
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

// SkipOnCloud skips the test if it's run on ClickHouse Cloud
func SkipOnCloud(t *testing.T, reasons ...string) {
	if CloudClickHouse {
		t.Skip(append(
			[]string{"Skipping test on cloud ClickHouse"},
			reasons...,
		))
	}
}

// SkipNotCloud skips the test if it's not run on ClickHouse Cloud
func SkipNotCloud(t *testing.T, reasons ...string) {
	if !CloudClickHouse {
		t.Skip(append(
			[]string{"Skipping test for non-cloud ClickHouse:"},
			reasons...,
		))
	}
}

// SkipOnHTTP skips the test if the protocol is HTTP
func SkipOnHTTP(t *testing.T, protocol clickhouse.Protocol, reasons ...string) {
	if protocol == clickhouse.HTTP {
		t.Skip(append(
			[]string{"Skipping HTTP test:"},
			reasons...,
		))
	}
}
