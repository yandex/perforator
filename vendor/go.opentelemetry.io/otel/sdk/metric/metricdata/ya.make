GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.40.0)

SRCS(
    data.go
    temporality.go
    temporality_string.go
)

END()

RECURSE(
    metricdatatest
)
