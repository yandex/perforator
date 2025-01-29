GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.0.0)

SRCS(
    doc.go
    write.go
)

GO_TEST_SRCS(write_test.go)

END()

RECURSE(
    gotest
)
