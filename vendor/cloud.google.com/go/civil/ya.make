GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.120.0)

SRCS(
    civil.go
)

GO_TEST_SRCS(civil_test.go)

END()

RECURSE(
    gotest
)
