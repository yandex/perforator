GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.0.1-0.20181226105442-5d4384ee4fb2)

SRCS(
    difflib.go
)

GO_TEST_SRCS(difflib_test.go)

END()

RECURSE(
    gotest
)
