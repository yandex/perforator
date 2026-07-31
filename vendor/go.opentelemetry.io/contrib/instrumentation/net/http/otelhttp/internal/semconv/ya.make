GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v0.67.0)

SRCS(
    client.go
    gen.go
    server.go
    util.go
)

GO_TEST_SRCS(
    bench_test.go
    client_test.go
    server_test.go
    util_test.go
)

GO_XTEST_SRCS(
    common_test.go
    httpconvtest_test.go
)

END()

RECURSE(
    gotest
)
