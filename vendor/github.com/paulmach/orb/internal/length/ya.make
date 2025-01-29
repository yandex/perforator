GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.11.1)

SRCS(
    length.go
)

GO_TEST_SRCS(length_test.go)

END()

RECURSE(
    gotest
)
