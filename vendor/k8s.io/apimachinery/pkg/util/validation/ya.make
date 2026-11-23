GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.37.2)

SRCS(
    ip.go
    validation.go
)

END()

RECURSE(
    field
)
