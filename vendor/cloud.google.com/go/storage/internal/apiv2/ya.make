GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.40.0)

SRCS(
    auxiliary.go
    doc.go
    storage_client.go
    version.go
)

GO_XTEST_SRCS(storage_client_example_test.go)

END()

RECURSE(
    gotest
    storagepb
)
