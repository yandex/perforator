GO_LIBRARY()

LICENSE(MIT)

VERSION(v2.0.1)

SRCS(
    chunkreader.go
)

GO_TEST_SRCS(chunkreader_test.go)

END()

RECURSE(
    gotest
)
