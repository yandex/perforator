GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.6.0)

SRCS(
    debug_disable.go
    diff.go
)

GO_TEST_SRCS(diff_test.go)

END()

RECURSE(
    gotest
)
