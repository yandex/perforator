GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    doc.go
    generate.go
)

END()

RECURSE(
    cli
    exec
    file
    http
    os
)
