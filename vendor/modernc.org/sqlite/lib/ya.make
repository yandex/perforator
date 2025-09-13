GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.37.0)

SRCS(
    defs.go
)

IF (ARCH_X86_64)
    SRCS(
        hooks.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_X86_64)
    SRCS(
        sqlite_linux_amd64.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM64)
    SRCS(
        hooks_linux_arm64.go
        sqlite_linux_arm64.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        hooks.go
        sqlite_linux_arm.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        mutex.go
    )
ENDIF()

IF (OS_DARWIN AND ARCH_X86_64)
    SRCS(
        sqlite_darwin_amd64.go
    )
ENDIF()

IF (OS_DARWIN AND ARCH_ARM64)
    SRCS(
        hooks.go
        sqlite_darwin_arm64.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        mutex.go
        sqlite_windows.go
    )
ENDIF()

IF (OS_WINDOWS AND ARCH_ARM64)
    SRCS(
        hooks.go
    )
ENDIF()

END()
