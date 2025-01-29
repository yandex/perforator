GO_LIBRARY()

SRCS(
    errors.go
)

IF (OS_LINUX)
    SRCS(
        mlock_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        mlock_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        mlock_other.go
    )
ENDIF()

END()
