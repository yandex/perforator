GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    ip.go
    validation.go
)

END()

RECURSE(
    field
)
