GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.8.2)

SRCS(
    package.go
    reader.go
    segment.go
)

GO_TEST_SRCS(reader_test.go)

END()

RECURSE(
    gotest
)
