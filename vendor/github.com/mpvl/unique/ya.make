GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.0.0-20150818121801-cbe035fff7de)

SRCS(
    unique.go
)

GO_TEST_SRCS(unique_test.go)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
