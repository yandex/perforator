GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.1.7)

SRCS(
    record.go
    ticketsender.go
)

GO_TEST_SRCS(
    record_test.go
    ticketsender_test.go
)

END()

RECURSE(
    gotest
    internal
)
