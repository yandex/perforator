GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.28.0)

SRCS(
    clock.go
    doc.go
    timeout.go
    writer.go
)

GO_TEST_SRCS(clock_test.go)

END()

RECURSE(
    gotest
)
