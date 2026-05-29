GO_LIBRARY()

SRCS(
    cli.go
    kv.go
    tskv.go
)

GO_TEST_SRCS(
    cli_test.go
    kv_test.go
    tskv_test.go
)

END()

RECURSE(
    gotest
    otelzap
)
