GO_LIBRARY()

LICENSE(MPL-2.0)

VERSION(v1.7.1)

GO_SKIP_TESTS(TestConnectorReturnsTimeout)

SRCS(
    atomic_bool.go
    auth.go
    buffer.go
    collations.go
    connection.go
    connector.go
    const.go
    driver.go
    dsn.go
    errors.go
    fields.go
    infile.go
    nulltime.go
    packets.go
    result.go
    rows.go
    statement.go
    transaction.go
    utils.go
)

GO_TEST_SRCS(
    auth_test.go
    benchmark_test.go
    connection_test.go
    connector_test.go
    driver_test.go
    dsn_test.go
    errors_test.go
    nulltime_test.go
    packets_test.go
    statement_test.go
    utils_test.go
)

IF (OS_LINUX)
    SRCS(
        conncheck.go
    )

    GO_TEST_SRCS(conncheck_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        conncheck.go
    )

    GO_TEST_SRCS(conncheck_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        conncheck_dummy.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
