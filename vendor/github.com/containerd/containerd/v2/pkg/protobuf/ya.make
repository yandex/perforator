GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.2.0)

SRCS(
    compare.go
    timestamp.go
)

END()

RECURSE(
    proto
    types
)
