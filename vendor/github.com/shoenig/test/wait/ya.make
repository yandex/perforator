GO_LIBRARY()

LICENSE(MPL-2.0)

VERSION(v1.7.1)

SRCS(
    wait.go
)

GO_TEST_SRCS(wait_test.go)

END()

RECURSE(
    gotest
)
