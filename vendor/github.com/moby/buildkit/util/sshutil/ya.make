GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.2)

SRCS(
    keyscan.go
    scpurl.go
)

GO_TEST_SRCS(scpurl_test.go)

END()

RECURSE(
    gotest
)
