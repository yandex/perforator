GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.35.1-0.20250728180453-01a3475a31bc)

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
    prometheus
)
