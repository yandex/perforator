GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.13.1)

SRCS(
    archive.go
)

GO_TEST_SRCS(archive_test.go)

END()

RECURSE(
    gotest
)
