GO_TEST_FOR(vendor/github.com/go-resty/resty/v2)

LICENSE(MIT)

VERSION(v2.15.3)

SIZE(MEDIUM)

# TEST_CWD(vendor/github.com/go-resty/resty/v2)

IF (NOT OPENSOURCE)
    # Removing the dependency on recipes

    DATA(
        arcadia/vendor/github.com/go-resty/resty/v2
    )

    DEPENDS(library/go/test/mutable_testdata)

    USE_RECIPE(
        library/go/test/mutable_testdata/mutable_testdata
        --testdata-dir
        vendor/github.com/go-resty/resty/v2
    )
ENDIF()

# This test is flaky

GO_SKIP_TESTS(TestClientRetryWaitCallback)

END()
