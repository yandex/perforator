GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.66.1)

SRCS(
    info.go
)

GO_TEST_SRCS(info_test.go)

END()

RECURSE(
    gotest
)
