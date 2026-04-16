GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

SRCS(
    controlplane.go
    e2e.go
    e2e_utils.go
)

GO_TEST_SRCS(e2e_test.go)

END()

RECURSE(
    gotest
)
