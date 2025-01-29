GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    pkg.go
    uuid.go
)

GO_XTEST_SRCS(uuid_test.go)

END()

RECURSE(
    gotest
)
