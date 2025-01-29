GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.1.7)

SRCS(
    ciphersuite.go
    counter.go
    expander.go
    halfconn.go
)

GO_TEST_SRCS(
    ciphersuite_test.go
    counter_test.go
    expander_test.go
    halfconn_test.go
)

END()

RECURSE(
    gotest
)
