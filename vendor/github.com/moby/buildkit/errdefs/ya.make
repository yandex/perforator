GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.2)

SRCS(
    internal.go
)

IF (OS_LINUX)
    SRCS(
        internal_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        internal_nolinux.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        internal_nolinux.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        internal_linux.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        internal_nolinux.go
    )
ENDIF()

END()
