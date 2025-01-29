GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    doc.go
    env.go
    pkg.go
)

GO_TEST_SRCS(env_test.go)

END()

RECURSE(
    gotest
)
