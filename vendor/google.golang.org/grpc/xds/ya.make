GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    server.go
    server_options.go
    xds.go
)

GO_TEST_SRCS(server_test.go)

GO_XTEST_SRCS(
    # server_ext_test.go
    # server_resource_ext_test.go
    # server_security_ext_test.go
    # server_serving_mode_ext_test.go
)

END()

RECURSE(
    bootstrap
    csds
    googledirectpath
    gotest
    internal
    test
)
