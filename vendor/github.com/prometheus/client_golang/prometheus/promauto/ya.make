GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.23.2)

SRCS(
    auto.go
)

GO_TEST_SRCS(auto_test.go)

END()

RECURSE(
    gotest
)
