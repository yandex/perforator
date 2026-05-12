GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    http.go
    interface.go
    port_range.go
    port_split.go
    util.go
)

END()

RECURSE(
    testing
)
