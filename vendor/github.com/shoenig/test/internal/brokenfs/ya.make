GO_LIBRARY()

LICENSE(MPL-2.0)

VERSION(v1.7.1)

IF (OS_LINUX)
    SRCS(
        fs_default.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        fs_default.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        fs_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        fs_default.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        fs_default.go
    )
ENDIF()

END()
