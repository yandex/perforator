GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    fault.go
)

GO_TEST_SRCS(
    # fault_test.go
)

END()

RECURSE(
    gotest
)
