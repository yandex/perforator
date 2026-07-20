GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.45.0)

SRCS(
    clone.go
    comment.go
    cursor.go
    equal.go
    fields.go
    purge.go
    stringlit.go
    unpack.go
    util.go
)

END()

RECURSE(
    free
)
