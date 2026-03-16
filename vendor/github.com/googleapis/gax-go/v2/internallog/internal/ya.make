GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.17.0)

SRCS(
    internal.go
)

END()

RECURSE(
    bookpb
    logtest
)
