GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    apply.go
    file.go
    resolve.go
    sanitize.go
    util.go
    walk.go
)

GO_TEST_SRCS(util_test.go)

GO_XTEST_SRCS(
    # apply_test.go
    # file_test.go
    # resolve_test.go
    # sanitize_test.go
)

END()

RECURSE(
    gotest
)
