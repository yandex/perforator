GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.20.2)

SRCS(
    grpcstats.go
    multi_span_exporter.go
    multispan.go
    tracing.go
)

END()

RECURSE(
    otlptracegrpc
)
