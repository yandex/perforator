GO_LIBRARY()

LICENSE(MPL-2.0)

VERSION(v1.1.0)

SRCS(
    errwrap.go
)

GO_TEST_SRCS(errwrap_test.go)

END()

RECURSE(
    gotest
)
