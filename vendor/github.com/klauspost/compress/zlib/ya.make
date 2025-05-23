GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause AND
    MIT
)

VERSION(v1.18.0)

SRCS(
    reader.go
    writer.go
)

GO_TEST_SRCS(
    # reader_test.go
    # writer_test.go
)

GO_XTEST_SRCS(
    # example_test.go
)

END()

RECURSE(
    gotest
)
