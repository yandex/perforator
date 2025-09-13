GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.35.1-0.20250728180453-01a3475a31bc)

SRCS(
    doc.go
    event.go
)

END()

RECURSE(
    core
    export
    keys
    label
)
