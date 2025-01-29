GO_TEST_FOR(vendor/github.com/aws/aws-sdk-go/aws/credentials/processcreds)

LICENSE(Apache-2.0)

VERSION(v1.46.7)

DATA(
    arcadia/vendor/github.com/aws/aws-sdk-go/aws/credentials/processcreds/testdata
)

TEST_CWD(vendor/github.com/aws/aws-sdk-go/aws/credentials/processcreds)

GO_SKIP_TESTS(
    TestProcessProviderAltConstruct
    TestProcessProviderExpectErrors
    TestProcessProviderExpired
    TestProcessProviderForceExpire
    TestProcessProviderFromSessionCfg
    TestProcessProviderFromSessionCrd
    TestProcessProviderFromSessionWithProfileCfg
    TestProcessProviderFromSessionWithProfileCrd
    TestProcessProviderNotExpired
    TestProcessProviderStatic
    TestProcessProviderTimeout
    TestProcessProviderWithLongSessionToken
)

END()
