GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.4.0)

SRCS(
    doc.go
    types.go
)

GO_TEST_SRCS(types_test.go)

END()

RECURSE(
    gotest
)
