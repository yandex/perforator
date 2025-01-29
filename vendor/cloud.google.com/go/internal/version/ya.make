GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.112.2)

SRCS(
    version.go
)

GO_TEST_SRCS(version_test.go)

END()

RECURSE(
    gotest
)
