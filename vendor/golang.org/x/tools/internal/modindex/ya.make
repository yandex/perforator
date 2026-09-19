GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.49.0)

SRCS(
    directories.go
    index.go
    lookup.go
    modindex.go
    symbols.go
)

END()

RECURSE(
    gomodindex
)
