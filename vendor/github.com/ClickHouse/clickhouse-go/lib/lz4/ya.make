GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.5.4)

SRCS(
    doc.go
    reader.go
    writer.go
)

GO_TEST_SRCS(lz4_test.go)

END()

RECURSE(
    gotest
)
