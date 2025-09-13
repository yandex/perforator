GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.5.2)

SRCS(
    call.go
    callset.go
    controller.go
    doc.go
    matchers.go
    string.go
)

END()

RECURSE(
    internal
)
