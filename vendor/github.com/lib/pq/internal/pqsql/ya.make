GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.12.0)

SRCS(
    copy.go
)

GO_TEST_SRCS(copy_test.go)

END()

RECURSE(
    gotest
)
