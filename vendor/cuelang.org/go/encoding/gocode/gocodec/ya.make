GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    codec.go
)

GO_TEST_SRCS(codec_test.go)

END()

RECURSE(
    gotest
)
