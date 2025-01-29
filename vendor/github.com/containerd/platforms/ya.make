GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.2.1)

SRCS(
    compare.go
    cpuinfo.go
    database.go
    defaults.go
    errors.go
    platforms.go
)

GO_TEST_SRCS(
    compare_test.go
    platforms_test.go
)

IF (OS_LINUX)
    SRCS(
        cpuinfo_linux.go
        defaults_unix.go
        platforms_other.go
    )

    GO_TEST_SRCS(
        cpuinfo_linux_test.go
        defaults_unix_test.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        cpuinfo_other.go
        defaults_darwin.go
        platforms_other.go
    )

    GO_TEST_SRCS(defaults_unix_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        cpuinfo_other.go
        defaults_windows.go
        platform_compat_windows.go
        platforms_windows.go
    )

    GO_TEST_SRCS(
        defaults_windows_test.go
        platform_compat_windows_test.go
        platforms_windows_test.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
