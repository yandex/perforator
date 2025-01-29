GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.40.0)

SRCS(
    version.go
)

END()

RECURSE(
    apiv2
    test
)
