GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.2.1)

SRCS(
    iterator.go
    message.go
    repeated.go
    scalar.go
)

GO_TEST_SRCS(
    example_group_test.go
    iterator_test.go
    message_test.go
    repeated_test.go
    scalar_test.go
)

GO_XTEST_SRCS(example_count_test.go)

END()

RECURSE(
    gotest
    internal
)
