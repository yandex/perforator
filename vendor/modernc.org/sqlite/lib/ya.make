GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.29.8)

SRCS(
    defs.go
    hooks.go
    mutex.go
)

IF (OS_LINUX AND ARCH_X86_64)
    SRCS(
        sqlite_linux_amd64.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM64)
    SRCS(
        sqlite_linux_arm64.go
    )
ENDIF()

IF (OS_DARWIN AND ARCH_X86_64)
    SRCS(
        sqlite_darwin_amd64.go
    )
ENDIF()

IF (OS_DARWIN AND ARCH_ARM64)
    SRCS(
        sqlite_darwin_arm64.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        sqlite_windows.go
    )
ENDIF()

END()
