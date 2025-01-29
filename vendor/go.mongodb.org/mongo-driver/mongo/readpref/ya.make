GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    mode.go
    options.go
    readpref.go
)

GO_TEST_SRCS(
    # mode_test.go
    # readpref_test.go
)

GO_XTEST_SRCS(options_example_test.go)

END()

RECURSE(
    gotest
)
