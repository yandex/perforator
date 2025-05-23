GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.217.0)

SRCS(
    dial.go
)

GO_TEST_SRCS(
    # dial_test.go
)

END()

RECURSE(
    gotest
)
