GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.0)

SRCS(
    call.go
    callset.go
    controller.go
    doc.go
    matchers.go
)

END()

RECURSE(
    internal
)
