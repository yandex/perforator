GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.5.4)

SRCS(
    remap.go
)

GO_TEST_SRCS(remap_test.go)

END()

RECURSE(
    gotest
)
