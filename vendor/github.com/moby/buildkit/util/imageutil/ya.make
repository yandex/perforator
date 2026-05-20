GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.2)

SRCS(
    config.go
    schema1.go
)

GO_TEST_SRCS(
    config_test.go
    schema1_test.go
)

END()

RECURSE(
    gotest
)
