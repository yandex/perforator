GO_TEST()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

GO_XTEST_SRCS(
    authority_test.go
    dump_test.go
    helpers_test.go
    lds_watchers_test.go
    metrics_test.go
    misc_watchers_test.go
)

END()
