GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.6.0)

SRCS(
    streams.go
)

GO_TEST_SRCS(streams_test.go)

END()

RECURSE(
    gotest
)
