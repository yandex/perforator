GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    position.go
    token.go
)

GO_TEST_SRCS(position_test.go)

END()

RECURSE(
    gotest
)
