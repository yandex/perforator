GO_TEST_FOR(vendor/github.com/aws/aws-sdk-go/aws/credentials)

LICENSE(Apache-2.0)

VERSION(v1.46.7)

GO_SKIP_TESTS(
    TestSharedCredentialsProvider
    TestSharedCredentialsProviderColonInCredFile
    TestSharedCredentialsProviderIsExpired
    TestSharedCredentialsProviderWithAWS_PROFILE
    TestSharedCredentialsProviderWithAWS_SHARED_CREDENTIALS_FILE
    TestSharedCredentialsProviderWithAWS_SHARED_CREDENTIALS_FILEAbsPath
    TestSharedCredentialsProviderWithoutTokenFromProfile
)

END()
