GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.14.1)

SRCS(
    internal.go
)

END()

RECURSE(
    bookpb
    logtest
)
