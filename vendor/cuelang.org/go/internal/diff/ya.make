GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    diff.go
    print.go
)

GO_TEST_SRCS(diff_test.go)

END()

RECURSE(
    gotest
)
