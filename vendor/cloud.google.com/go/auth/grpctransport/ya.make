GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

GO_SKIP_TESTS(TestWithEndpointAndPoolSize)

DATA(
    arcadia/vendor/cloud.google.com/go/auth/internal/testdata
)

TEST_CWD(vendor/cloud.google.com/go/auth/grpctransport)

SRCS(
    directpath.go
    grpctransport.go
    pool.go
)

GO_TEST_SRCS(
    grpctransport_test.go
    pool_test.go
)

IF (OS_LINUX)
    SRCS(
        dial_socketopt.go
    )

    GO_TEST_SRCS(dial_socketopt_test.go)
ENDIF()

END()

RECURSE(
    gotest
    testdata
)
