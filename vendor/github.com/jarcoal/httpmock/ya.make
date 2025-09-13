GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.3.1)

SRCS(
    doc.go
    env.go
    file.go
    match.go
    response.go
    transport.go
)

GO_TEST_SRCS(export_test.go)

GO_XTEST_SRCS(
    env_test.go
    file_test.go
    match_test.go
    race_test.go
    response_test.go
    transport_test.go
    util_test.go
)

END()

RECURSE(
    gotest
    internal
)
