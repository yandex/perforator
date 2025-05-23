GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.33.1)

SRCS(
    string.go
)

IF (ARCH_X86_64)
    SRCS(
        string_unsafe.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        string_unsafe.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        string_safe.go
    )
ENDIF()

END()
