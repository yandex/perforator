GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.60.0)

SRCS(
    experimental.go
    version.go
)

END()

RECURSE(
    apiv2
    test
)
