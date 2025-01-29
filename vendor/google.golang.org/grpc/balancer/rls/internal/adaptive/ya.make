GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    adaptive.go
    lookback.go
)

GO_TEST_SRCS(
    adaptive_test.go
    lookback_test.go
)

END()

RECURSE(
    gotest
)
