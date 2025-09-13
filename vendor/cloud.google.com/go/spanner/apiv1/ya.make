GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.76.1)

SRCS(
    auxiliary.go
    auxiliary_go123.go
    doc.go
    helpers.go
    info.go
    path_funcs.go
    spanner_client.go
    spanner_client_options.go
    version.go
)

GO_XTEST_SRCS(
    spanner_client_example_go123_test.go
    spanner_client_example_test.go
)

END()

RECURSE(
    gotest
    spannerpb
)
