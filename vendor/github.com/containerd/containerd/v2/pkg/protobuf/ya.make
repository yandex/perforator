GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.1.5)

SRCS(
    compare.go
    timestamp.go
)

END()

RECURSE(
    proto
    types
)
