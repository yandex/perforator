GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.22.1-0.20240829175637-39126e24d653)

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
