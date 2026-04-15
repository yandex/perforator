GO_PROGRAM()

LICENSE(BSD-3-Clause)

VERSION(v0.51.0)

IF (ARCH_X86_64)
    SRCS(
        h2i.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        h2i.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        h2i.go
    )
ENDIF()

END()
