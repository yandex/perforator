GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v4.0.5)

SRCS(
    decode.go
    encode.go
    indent.go
    scanner.go
    stream.go
    tags.go
)

GO_TEST_SRCS(
    bench_test.go
    decode_test.go
    encode_test.go
    number_test.go
    scanner_test.go
    stream_test.go
    tagkey_test.go
    tags_test.go
)

END()

RECURSE(
    gotest
)
