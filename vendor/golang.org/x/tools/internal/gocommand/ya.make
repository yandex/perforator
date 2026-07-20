GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.45.0)

SRCS(
    invoke.go
    vendor.go
    version.go
)

IF (OS_LINUX)
    SRCS(
        invoke_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        invoke_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        invoke_notunix.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        invoke_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        invoke_notunix.go
    )
ENDIF()

END()
