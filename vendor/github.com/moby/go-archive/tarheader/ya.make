GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.2.0)

SRCS(
    tarheader.go
)

IF (OS_LINUX)
    SRCS(
        tarheader_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        tarheader_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        tarheader_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        tarheader_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        tarheader_unix.go
    )
ENDIF()

END()
