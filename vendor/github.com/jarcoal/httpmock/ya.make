GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.0.8)

SRCS(
    doc.go
    env.go
    file.go
    response.go
    transport.go
)

GO_TEST_SRCS(env_test.go)

GO_XTEST_SRCS(
    file_test.go
    response_test.go
    transport_test.go
    util_test.go
)

END()

RECURSE(
    gotest
    internal
)
