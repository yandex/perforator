GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.37.2)

SRCS(
    buffers.go
    decode.go
    diagnostic.go
    encode.go
    transcoding.go
)

GO_TEST_SRCS(
    buffers_test.go
    transcoding_test.go
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
