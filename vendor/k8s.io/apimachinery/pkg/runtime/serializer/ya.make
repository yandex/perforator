GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    codec_factory.go
    negotiated_codec.go
)

GO_TEST_SRCS(
    codec_test.go
    encoder_with_allocator_test.go
    sparse_test.go
)

END()

RECURSE(
    gotest
    json
    protobuf
    recognizer
    streaming
    versioning
    yaml
)
