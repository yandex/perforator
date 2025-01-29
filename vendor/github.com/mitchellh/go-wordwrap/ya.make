GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.0.1)

SRCS(
    wordwrap.go
)

GO_TEST_SRCS(wordwrap_test.go)

END()

RECURSE(
    gotest
)
