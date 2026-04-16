GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

SRCS(
    doc.go
    file_reader.go
    token_file_call_creds.go
)

GO_TEST_SRCS(
    file_reader_test.go
    token_file_call_creds_test.go
)

END()

RECURSE(
    gotest
)
