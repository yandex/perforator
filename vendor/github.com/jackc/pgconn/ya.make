GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.14.0)

SRCS(
    auth_scram.go
    config.go
    doc.go
    errors.go
    krb5.go
    pgconn.go
)

GO_TEST_SRCS(export_test.go)

GO_XTEST_SRCS(
    benchmark_test.go
    config_test.go
    errors_test.go
    frontend_test.go
    helper_test.go
    # pgconn_stress_test.go
    # pgconn_test.go
)

IF (OS_LINUX)
    SRCS(
        defaults.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        defaults.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        defaults_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
    internal
    stmtcache
)
