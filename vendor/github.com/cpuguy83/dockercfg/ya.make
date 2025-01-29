GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.3.1)

SRCS(
    auth.go
    config.go
    load.go
)

GO_TEST_SRCS(auth_test.go)

END()

RECURSE(
    gotest
)
