GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.12.0)

SRCS(
    proto.go
)

IF (ARCH_X86_64)
    SRCS(
        sz_64.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        sz_64.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        sz_32.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        sz_64.go
    )
ENDIF()

END()
