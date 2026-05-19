GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.0.1)

SRCS(
    locker.go
)

GO_TEST_SRCS(locker_test.go)

END()

RECURSE(
    gotest
)
