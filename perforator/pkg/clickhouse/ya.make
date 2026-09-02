GO_LIBRARY()

SRCS(
    clickhouse.go
    exec_retry.go
)

GO_TEST_SRCS(exec_retry_test.go)

END()

RECURSE(gotest)
