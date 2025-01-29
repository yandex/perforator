GO_TEST_FOR(vendor/github.com/aws/aws-sdk-go/aws/session)

LICENSE(Apache-2.0)

VERSION(v1.46.7)

DATA(
    arcadia/vendor/github.com/aws/aws-sdk-go/aws/session/testdata
)

TEST_CWD(vendor/github.com/aws/aws-sdk-go/aws/session)

GO_SKIP_TESTS(TestSharedConfigCredentialSource)

TAG(ya:go_total_report)

END()
