GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    idle.go
)

GO_TEST_SRCS(idle_test.go)

GO_XTEST_SRCS(
    # idle_e2e_test.go
)

END()

RECURSE(
    gotest
)
