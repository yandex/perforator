GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.35.1-0.20250728180453-01a3475a31bc)

SRCS(
    emptyvfs.go
    fs.go
    namespace.go
    os.go
    vfs.go
)

END()

RECURSE(
    gatefs
    httpfs
    mapfs
    zipfs
)
