GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.9.4)

IF (OS_LINUX)
    SRCS(
        syslog.go
    )

    GO_TEST_SRCS(syslog_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        syslog.go
    )

    GO_TEST_SRCS(syslog_test.go)
ENDIF()

IF (OS_ANDROID)
    SRCS(
        syslog.go
    )

    GO_TEST_SRCS(syslog_test.go)
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        syslog.go
    )

    GO_TEST_SRCS(syslog_test.go)
ENDIF()

END()

RECURSE(
    gotest
)
