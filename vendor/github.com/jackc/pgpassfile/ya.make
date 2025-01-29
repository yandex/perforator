GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.0.0)

SRCS(
    pgpass.go
)

GO_TEST_SRCS(pgpass_test.go)

END()

RECURSE(
    gotest
)
