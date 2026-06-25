GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.3)

SRCS(
    doc.go
    error.go
)

IF (OS_LINUX)
    SRCS(
        errno_linux.go
        types_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        errno_other.go
        strings_other.go
        types_other.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        errno_string_windows.go
        errno_windows.go
        strings_windows.go
        types_other.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        errno_linux.go
        types_linux.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        errno_other.go
        strings_other.go
        types_other.go
    )
ENDIF()

END()
