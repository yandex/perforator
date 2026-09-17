GO_TEST_FOR(vendor/github.com/jackc/pgconn)

LICENSE(MIT)

VERSION(v1.14.3)

GO_SKIP_TESTS(
    TestConfigCopyCanBeUsedToConnect
    TestFrontendFatalErrExec
)

END()
