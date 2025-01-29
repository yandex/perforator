GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    error.go
    reader.go
    shared.go
    signer.go
    stream_writer.go
    transport.go
    writer.go
)

GO_TEST_SRCS(
    reader_test.go
    shared_test.go
    signer_test.go
    writer_test.go
)

END()

RECURSE(
    gotest
)
