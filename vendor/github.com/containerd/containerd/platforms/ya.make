GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.7.29)

SRCS(
    platforms_deprecated.go
)

IF (OS_LINUX)
    SRCS(
        platforms_deprecated_other.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        platforms_deprecated_other.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        platforms_deprecated_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        platforms_deprecated_other.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        platforms_deprecated_other.go
    )
ENDIF()

END()
