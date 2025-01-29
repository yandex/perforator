GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    opencensus.go
)

END()

RECURSE(
    internal
    metric
    plugin
    resource
    stats
    tag
    trace
)
