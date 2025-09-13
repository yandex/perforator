GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.27.0)

SRCS(
    tag.go
)

GO_TEST_SRCS(tag_test.go)

END()

RECURSE(
    gotest
)
