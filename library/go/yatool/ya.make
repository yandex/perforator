GO_LIBRARY()

SRCS(
    root.go
    ya.go
)

GO_XTEST_SRCS(
    root_example_test.go
    root_test.go
    suite_test.go
    ya_example_test.go
    ya_test.go
)

END()

RECURSE(
    gotest
)
