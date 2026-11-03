GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.34.3)

SRCS(
    ip.go
    validation.go
)

END()

RECURSE(
    field
)
