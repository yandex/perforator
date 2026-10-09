GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

ALL_GO_TEST_SRCS()

SRCS(
    rbac.go
)

END()

RECURSE(
    gotest
)
