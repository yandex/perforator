GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.80.0)

SRCS(
    pluginoption.go
)

END()

RECURSE(
    testutils
    tracing
)
