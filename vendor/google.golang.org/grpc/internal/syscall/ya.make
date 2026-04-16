GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

IF (OS_LINUX)
    SRCS(
        syscall_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        syscall_nonlinux.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        syscall_nonlinux.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        syscall_linux.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        syscall_nonlinux.go
    )
ENDIF()

END()
