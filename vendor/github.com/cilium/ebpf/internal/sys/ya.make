GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.3)

SRCS(
    doc.go
    fd.go
    pinning.go
    ptr.go
    signals.go
    syscall.go
    types.go
)

IF (ARCH_X86_64)
    SRCS(
        ptr_64.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        ptr_64.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        ptr_32_le.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        ptr_64.go
    )
ENDIF()

END()
