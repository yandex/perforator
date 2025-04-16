GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.215.0)

SRCS(
    option.go
)

GO_TEST_SRCS(
    # option_test.go
)

END()

RECURSE(
    gotest
    internaloption
)
