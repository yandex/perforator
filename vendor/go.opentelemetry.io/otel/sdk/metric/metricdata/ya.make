GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.36.0)

SRCS(
    data.go
    temporality.go
    temporality_string.go
)

END()

RECURSE(
    metricdatatest
)
