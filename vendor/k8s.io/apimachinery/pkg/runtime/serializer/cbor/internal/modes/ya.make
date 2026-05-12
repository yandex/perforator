GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    buffers.go
    custom.go
    decode.go
    diagnostic.go
    encode.go
)

GO_TEST_SRCS(
    buffers_test.go
    custom_test.go
)

GO_XTEST_SRCS(
    appendixa_test.go
    decode_test.go
    encode_test.go
    modes_test.go
    roundtrip_test.go
)

END()

RECURSE(
    gotest
)
