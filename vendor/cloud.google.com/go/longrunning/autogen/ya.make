GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.6.5)

SRCS(
    auxiliary.go
    auxiliary_go123.go
    doc.go
    from_conn.go
    helpers.go
    info.go
    operations_client.go
)

GO_XTEST_SRCS(
    operations_client_example_go123_test.go
    operations_client_example_test.go
)

END()

RECURSE(
    gotest
    longrunningpb
)
