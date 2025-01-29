GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.176.1)

GO_SKIP_TESTS(
    TestLogDirectPathMisconfigAttrempDirectPathNotSet
    TestLogDirectPathMisconfigNotOnGCE
)

SRCS(
    dial.go
    pool.go
)

GO_TEST_SRCS(
    dial_test.go
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
)
