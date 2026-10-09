GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

SRCS(
    pluginoption.go
)

END()

RECURSE(
    testutils
    tracing
)
