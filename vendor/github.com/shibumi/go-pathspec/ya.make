GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.3.0)

SRCS(
    gitignore.go
)

GO_TEST_SRCS(gitignore_test.go)

END()

RECURSE(
    gotest
)
