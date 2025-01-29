GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.11.1)

SRCS(
    define.go
    helpers.go
    projections.go
)

GO_TEST_SRCS(
    define_test.go
    helpers_test.go
    projections_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
