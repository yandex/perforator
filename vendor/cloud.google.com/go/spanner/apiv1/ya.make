GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.60.0)

SRCS(
    auxiliary.go
    doc.go
    info.go
    path_funcs.go
    spanner_client.go
    spanner_client_options.go
    version.go
)

GO_XTEST_SRCS(spanner_client_example_test.go)

END()

RECURSE(
    gotest
    spannerpb
)
