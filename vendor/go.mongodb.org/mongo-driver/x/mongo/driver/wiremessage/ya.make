GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    wiremessage.go
)

GO_TEST_SRCS(wiremessage_test.go)

END()

RECURSE(
    gotest
)
