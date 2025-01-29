GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.21.0)

SRCS(
    code.go
    gen.go
)

END()

RECURSE(
    bitfield
)
