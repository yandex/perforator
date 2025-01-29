GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.1.2)

SRCS(
    client.go
    client_conv.go
    common.go
    doc.go
    parse.go
    scram.go
    server.go
    server_conv.go
)

GO_TEST_SRCS(
    client_conv_test.go
    common_test.go
    server_conv_test.go
    testdata_test.go
)

GO_XTEST_SRCS(doc_test.go)

END()

RECURSE(
    gotest
)
