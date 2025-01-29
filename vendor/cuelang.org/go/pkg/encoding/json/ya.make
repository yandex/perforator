GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    json.go
    manual.go
    pkg.go
)

GO_XTEST_SRCS(json_test.go)

END()

RECURSE(
    gotest
)
