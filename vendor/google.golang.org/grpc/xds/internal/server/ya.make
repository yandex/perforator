GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    conn_wrapper.go
    listener_wrapper.go
    rds_handler.go
)

GO_TEST_SRCS(rds_handler_test.go)

END()

RECURSE(
    gotest
)
