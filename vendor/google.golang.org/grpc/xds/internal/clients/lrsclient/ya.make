GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    load_store.go
    logging.go
    lrs_stream.go
    lrsclient.go
    lrsconfig.go
)

GO_TEST_SRCS(load_store_test.go)

GO_XTEST_SRCS(loadreport_test.go)

END()

RECURSE(
    gotest
)
