GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    manual.go
    pkg.go
    strings.go
)

GO_XTEST_SRCS(strings_test.go)

END()

RECURSE(
    gotest
)
