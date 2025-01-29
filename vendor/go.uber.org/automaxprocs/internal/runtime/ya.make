GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.6.0)

SRCS(
    runtime.go
)

IF (OS_LINUX)
    SRCS(
        cpu_quota_linux.go
    )

    GO_TEST_SRCS(cpu_quota_linux_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        cpu_quota_unsupported.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        cpu_quota_unsupported.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
