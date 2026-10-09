GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.2)

DATA(
    arcadia/vendor/cloud.google.com/go/auth/internal/testdata
)

TEST_CWD(vendor/cloud.google.com/go/auth/credentials/idtoken)

GO_SKIP_TESTS(
    TestNewCredentials_CredentialsFile
    TestNewCredentials_CredentialsJSON
)

SRCS(
    cache.go
    compute.go
    file.go
    idtoken.go
    validate.go
)

GO_TEST_SRCS(
    cache_test.go
    compute_test.go
    idtoken_test.go
    validate_test.go
)

GO_XTEST_SRCS(
    examples_test.go
    integration_test.go
)

END()

RECURSE(
    gotest
)
