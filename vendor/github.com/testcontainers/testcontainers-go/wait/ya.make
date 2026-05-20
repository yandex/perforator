GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.32.0)

SRCS(
    all.go
    exec.go
    exit.go
    health.go
    host_port.go
    http.go
    log.go
    nop.go
    sql.go
    wait.go
)

GO_TEST_SRCS(
    all_test.go
    exit_test.go
    health_test.go
    host_port_test.go
    log_test.go
    sql_test.go
    wait_test.go
)

GO_XTEST_SRCS(
    # exec_test.go
    # http_test.go
)

IF (OS_LINUX)
    SRCS(
        errors.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        errors.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        errors_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        errors.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        errors.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
