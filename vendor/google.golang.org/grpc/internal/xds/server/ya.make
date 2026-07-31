GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.80.0)

SRCS(
    conn_wrapper.go
    filter_chain_manager.go
    listener_wrapper.go
    rds_handler.go
    routing.go
)

GO_TEST_SRCS(
    # filter_chain_manager_test.go
    # rds_handler_test.go
    # routing_test.go
)

END()

RECURSE(
    gotest
)
