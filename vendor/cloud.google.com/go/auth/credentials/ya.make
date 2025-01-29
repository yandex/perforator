GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

DATA(
    arcadia/vendor/cloud.google.com/go/auth/internal/testdata
)

TEST_CWD(vendor/cloud.google.com/go/auth/credentials)

SRCS(
    compute.go
    detect.go
    doc.go
    filetypes.go
    selfsignedjwt.go
)

GO_TEST_SRCS(
    compute_test.go
    detect_test.go
    selfsignedjwt_test.go
)

GO_XTEST_SRCS(
    # example_test.go
)

END()

RECURSE(
    downscope
    externalaccount
    gotest
    idtoken
    impersonate
    internal
)
