GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    api.go
    doc.go
    errors.go
    service.go
)

END()

RECURSE(
    ssoiface
)
