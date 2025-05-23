GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v4.1.22)

SRCS(
    xxh32zero.go
)

IF (ARCH_X86_64)
    SRCS(
        xxh32zero_other.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        xxh32zero_other.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        xxh32zero_arm.go
        xxh32zero_arm.s
    )
ENDIF()

END()
