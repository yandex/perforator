GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.63.0)

SRCS(
    info.go
)

GO_TEST_SRCS(info_test.go)

END()

RECURSE(
    gotest
)
