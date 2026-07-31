GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.42.0)

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
