GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.9.21)

SRCS(
    config.go
    devicetoken.go
    persist.go
    sender.go
    token.go
    token_1.13.go
    version.go
)

GO_TEST_SRCS(
    config_test.go
    devicetoken_test.go
    persist_test.go
    token_test.go
)

END()

RECURSE(
    cmd
    gotest
)
