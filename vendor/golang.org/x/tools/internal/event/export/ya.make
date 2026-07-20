GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.45.0)

SRCS(
    id.go
    labels.go
    log.go
    printer.go
    trace.go
)

END()

RECURSE(
    eventtest
    metric
    otel
    prometheus
)
