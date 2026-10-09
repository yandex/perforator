GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

SRCS(
    profiling.go
)

END()

RECURSE(
    cmd
    proto
    service
)
