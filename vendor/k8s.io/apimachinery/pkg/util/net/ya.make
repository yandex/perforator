GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    http.go
    interface.go
    port_range.go
    port_split.go
    util.go
)

GO_TEST_SRCS(
    http_test.go
    interface_test.go
    port_range_test.go
    port_split_test.go
    util_test.go
)

END()

RECURSE(
    gotest
)
