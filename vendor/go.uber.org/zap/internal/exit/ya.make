GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.28.0)

SRCS(
    exit.go
)

GO_XTEST_SRCS(exit_test.go)

END()

RECURSE(
    gotest
)
