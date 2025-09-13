GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.35.0)

SRCS(
    envconfig.go
    options.go
    optiontypes.go
    tls.go
)

GO_TEST_SRCS(options_test.go)

END()

RECURSE(
    gotest
)
