GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.35.1-0.20250728180453-01a3475a31bc)

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
