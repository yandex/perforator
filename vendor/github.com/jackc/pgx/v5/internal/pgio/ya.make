GO_LIBRARY()

LICENSE(MIT)

VERSION(v5.7.1)

SRCS(
    doc.go
    write.go
)

GO_TEST_SRCS(write_test.go)

END()

RECURSE(
    gotest
)
