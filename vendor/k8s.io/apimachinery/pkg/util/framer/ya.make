GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    framer.go
)

GO_TEST_SRCS(framer_test.go)

END()

RECURSE(
    gotest
)
