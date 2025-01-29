GO_LIBRARY()

LICENSE(MIT)

VERSION(v5.7.1)

SRCS(
    pgmock.go
)

GO_XTEST_SRCS(pgmock_test.go)

END()

RECURSE(
    gotest
)
