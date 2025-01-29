GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    cuecontext.go
)

GO_TEST_SRCS(cuecontext_test.go)

END()

RECURSE(
    gotest
)
