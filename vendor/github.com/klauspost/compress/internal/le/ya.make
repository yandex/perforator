GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause AND
    MIT
)

VERSION(v1.18.0)

SRCS(
    le.go
)

IF (ARCH_X86_64)
    SRCS(
        unsafe_enabled.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        unsafe_enabled.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM7)
    SRCS(
        unsafe_disabled.go
    )
ENDIF()

END()
