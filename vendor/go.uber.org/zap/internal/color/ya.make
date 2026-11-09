GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.28.0)

SRCS(
    color.go
)

GO_TEST_SRCS(color_test.go)

END()

RECURSE(
    gotest
)
