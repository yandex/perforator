GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.51.0)

SRCS(
    auxiliary.go
    auxiliary_go123.go
    doc.go
    helpers.go
    storage_client.go
    version.go
)

GO_XTEST_SRCS(
    storage_client_example_go123_test.go
    storage_client_example_test.go
)

END()

RECURSE(
    gotest
    storagepb
)
