GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.36.0)

SRCS(
    reuse_slice.go
)

END()

RECURSE(
    aggregate
    x
)
