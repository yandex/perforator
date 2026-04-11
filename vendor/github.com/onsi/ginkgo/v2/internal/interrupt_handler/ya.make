GO_LIBRARY()

LICENSE(MIT)

VERSION(v2.23.4)

SRCS(
    interrupt_handler.go
)

IF (OS_LINUX)
    SRCS(
        sigquit_swallower_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        sigquit_swallower_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        sigquit_swallower_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        sigquit_swallower_unix.go
    )
ENDIF()

END()
