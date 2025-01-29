GO_TEST_FOR(vendor/golang.org/x/text/language)

LICENSE(BSD-3-Clause)

VERSION(v0.21.0)

DATA(
    arcadia/vendor/golang.org/x/text/language/testdata
)

TEST_CWD(vendor/golang.org/x/text/language)

GO_SKIP_TESTS(TestCompliance)

END()
