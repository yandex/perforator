GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

DATA(
    arcadia/vendor/cloud.google.com/go/auth/internal/testdata
)

TEST_CWD(vendor/cloud.google.com/go/auth/internal/credsfile)

SRCS(
    credsfile.go
    filetype.go
    parse.go
)

GO_TEST_SRCS(parse_test.go)

END()

RECURSE(
    gotest
)
