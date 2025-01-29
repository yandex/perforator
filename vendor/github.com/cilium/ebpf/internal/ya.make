GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.1)

SRCS(
    buffer.go
    deque.go
    elf.go
    endian_le.go
    errors.go
    feature.go
    io.go
    math.go
    output.go
    prog.go
    version.go
)

END()

RECURSE(
    cmd
    epoll
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
