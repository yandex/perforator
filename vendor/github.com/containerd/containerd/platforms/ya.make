GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.7.20)

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

END()
