GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    chunker.go
    cuetest.go
    nolong.go
)

GO_TEST_SRCS(cuetest_test.go)

END()

RECURSE(
    gotest
)
