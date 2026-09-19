GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.42.0)

SRCS(
    internal.go
)

END()

RECURSE(
    enctest
    identifier
)
