GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    builder.go
)

GO_TEST_SRCS(builder_test.go)

END()

RECURSE(
    gotest
)
