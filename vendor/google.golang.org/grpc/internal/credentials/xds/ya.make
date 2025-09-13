GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    handshake_info.go
)

GO_TEST_SRCS(handshake_info_test.go)

END()

RECURSE(
    gotest
)
