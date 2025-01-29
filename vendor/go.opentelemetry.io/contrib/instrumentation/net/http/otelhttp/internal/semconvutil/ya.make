GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.57.0)

SRCS(
    gen.go
    httpconv.go
    netconv.go
)

GO_TEST_SRCS(
    httpconv_test.go
    netconv_test.go
)

END()

RECURSE(
    gotest
)
