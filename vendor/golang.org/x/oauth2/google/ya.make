GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.30.0)

SRCS(
    appengine.go
    default.go
    doc.go
    error.go
    google.go
    jwt.go
    sdk.go
)

END()

RECURSE(
    downscope
    externalaccount
    internal
)
