GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v28.0.4+incompatible)

SRCS(
    idtools.go
)

IF (OS_LINUX)
    SRCS(
        idtools_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        idtools_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        idtools_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        idtools_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        idtools_unix.go
    )
ENDIF()

END()
