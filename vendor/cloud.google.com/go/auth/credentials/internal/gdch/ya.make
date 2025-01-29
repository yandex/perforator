GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

DATA(
    arcadia/vendor/cloud.google.com/go/auth/internal/testdata
)

TEST_CWD(vendor/cloud.google.com/go/auth/credentials/internal/gdch)

SRCS(
    gdch.go
)

GO_TEST_SRCS(gdch_test.go)

END()

RECURSE(
    gotest
)
