GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.9.4)

SRCS(
    writer.go
)

GO_TEST_SRCS(writer_test.go)

END()

RECURSE(
    gotest
)
