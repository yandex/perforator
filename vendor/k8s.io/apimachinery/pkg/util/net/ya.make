GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.37.2)

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
