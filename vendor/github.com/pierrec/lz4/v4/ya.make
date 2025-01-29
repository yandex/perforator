GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v4.1.21)

SRCS(
    compressing_reader.go
    lz4.go
    options.go
    options_gen.go
    reader.go
    state.go
    state_gen.go
    writer.go
)

END()

RECURSE(
    internal
)
