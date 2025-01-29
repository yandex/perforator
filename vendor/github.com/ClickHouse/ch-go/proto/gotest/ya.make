GO_TEST_FOR(vendor/github.com/ClickHouse/ch-go/proto)

LICENSE(Apache-2.0)

VERSION(v0.58.2)

GO_SKIP_TESTS(
    TestDump
    TestDumpLowCardinality
)

END()
