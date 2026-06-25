GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.3)

SRCS(
    buffer.go
    deque.go
    elf.go
    errors.go
    feature.go
    goos.go
    io.go
    math.go
    output.go
    prog.go
    version.go
)

IF (ARCH_X86_64)
    SRCS(
        endian_le.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        endian_le.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        endian_le.go
    )
ENDIF()

END()

RECURSE(
    cmd
    kallsyms
    kconfig
    linux
    sys
    sysenc
    testutils
    tracefs
)

IF (OS_LINUX)
    RECURSE(
        unix
        epoll
    )
ENDIF()

IF (OS_DARWIN)
    RECURSE(
        unix
    )
ENDIF()

IF (OS_WINDOWS)
    RECURSE(
        unix
    )
ENDIF()
