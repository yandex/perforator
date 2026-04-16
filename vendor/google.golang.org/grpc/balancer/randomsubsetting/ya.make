GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

SRCS(
    randomsubsetting.go
)

GO_TEST_SRCS(randomsubsetting_test.go)

END()

RECURSE(
    gotest
)
