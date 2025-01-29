GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    manual.go
    pkg.go
    regexp.go
)

GO_XTEST_SRCS(regexp_test.go)

END()

RECURSE(
    gotest
)
