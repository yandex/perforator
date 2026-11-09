GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.28.0)

SRCS(
    writer.go
)

GO_TEST_SRCS(writer_test.go)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
