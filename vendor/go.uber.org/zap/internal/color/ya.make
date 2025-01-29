GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.27.0)

SRCS(
    color.go
)

GO_TEST_SRCS(color_test.go)

END()

RECURSE(
    gotest
)
