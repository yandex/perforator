GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.36.0)

SRCS(
    env.go
)

GO_TEST_SRCS(env_test.go)

END()

RECURSE(
    gotest
)
