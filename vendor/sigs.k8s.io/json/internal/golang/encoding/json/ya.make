GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v0.0.0-20250730193827-2d320260d730)

SRCS(
    decode.go
    encode.go
    fold.go
    indent.go
    kubernetes_patch.go
    scanner.go
    stream.go
    tables.go
    tags.go
)

END()
