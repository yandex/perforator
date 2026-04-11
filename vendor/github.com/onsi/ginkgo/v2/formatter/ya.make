GO_LIBRARY()

LICENSE(MIT)

VERSION(v2.23.4)

SRCS(
    formatter.go
)

IF (OS_LINUX)
    SRCS(
        colorable_others.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        colorable_others.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        colorable_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        colorable_others.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        colorable_others.go
    )
ENDIF()

END()
