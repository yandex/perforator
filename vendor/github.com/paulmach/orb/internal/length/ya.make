GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.12.0)

SRCS(
    length.go
)

GO_TEST_SRCS(length_test.go)

END()

RECURSE(
    gotest
)
