GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.3.12)

SRCS(
    sysconf.go
)

IF (OS_LINUX)
    SRCS(
        sysconf_generic.go
        sysconf_linux.go
        sysconf_posix.go
        zsysconf_defs_linux.go
    )

    GO_TEST_SRCS(sysconf_linux_test.go)

    GO_XTEST_SRCS(
        example_test.go
        sysconf_test.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_X86_64)
    SRCS(
        zsysconf_values_linux_amd64.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM64)
    SRCS(
        zsysconf_values_linux_arm64.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        zsysconf_values_linux_arm.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        sysconf_bsd.go
        sysconf_darwin.go
        sysconf_generic.go
        sysconf_posix.go
        zsysconf_defs_darwin.go
    )

    GO_XTEST_SRCS(
        example_test.go
        sysconf_test.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        sysconf_unsupported.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
