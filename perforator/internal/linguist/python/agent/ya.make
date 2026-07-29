GO_LIBRARY()

GO_EMBED_PATTERN(offsets/*.json)

SRCS(
    offsets.go
    python.go
    version.go
)

GO_TEST_SRCS(python_test.go)

END()

RECURSE(
    linetable
)

RECURSE_FOR_TESTS(
    gotest
)
