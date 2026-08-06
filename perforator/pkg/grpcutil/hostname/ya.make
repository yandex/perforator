GO_LIBRARY()

SRCS(
    credentials.go
    interceptor.go
)

GO_TEST_SRCS(
    credentials_test.go
    interceptor_test.go
)

END()

RECURSE(
    gotest
)
