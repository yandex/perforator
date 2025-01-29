GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    manual.go
    pkg.go
)

GO_XTEST_SRCS(yaml_test.go)

END()

RECURSE(
    gotest
)
