GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    client_session.go
    cluster_clock.go
    doc.go
    options.go
    server_session.go
    session_pool.go
)

GO_TEST_SRCS(
    client_session_test.go
    cluster_clock_test.go
    server_session_test.go
    session_pool_test.go
)

END()

RECURSE(
    gotest
)
