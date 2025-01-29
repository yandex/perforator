GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.6.0)

SRCS(
    maxprocs.go
    version.go
)

GO_TEST_SRCS(maxprocs_test.go)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
