GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.5.0)

SRCS(
    charset_backcompat_deny.go
    errors.go
    id.go
    match.go
    path.go
    require.go
    trustdomain.go
)

GO_TEST_SRCS(path_test.go)

GO_XTEST_SRCS(
    id_test.go
    match_test.go
    require_test.go
    trustdomain_test.go
)

END()

RECURSE(
    gotest
)
