GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.0.1)

SRCS(
    words.go
)

GO_TEST_SRCS(words_test.go)

END()

RECURSE(
    gotest
)
