GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.5.0)

SRCS(
    duration.go
    size.go
    ulimit.go
)

GO_TEST_SRCS(
    duration_test.go
    size_test.go
    ulimit_test.go
)

END()

RECURSE(
    gotest
)
