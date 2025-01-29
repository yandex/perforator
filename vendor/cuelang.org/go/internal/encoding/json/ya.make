GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    encode.go
)

GO_TEST_SRCS(encode_test.go)

END()

RECURSE(
    gotest
)
