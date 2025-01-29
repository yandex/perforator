GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

SRCS(
    jwt.go
)

GO_TEST_SRCS(jwt_test.go)

END()

RECURSE(
    gotest
)
