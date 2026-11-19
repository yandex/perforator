GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.4.4)

SRCS(
    funcr.go
    slogsink.go
)

END()

RECURSE(
    example
)
