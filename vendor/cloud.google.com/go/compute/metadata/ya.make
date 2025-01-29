GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.5.0)

SRCS(
    metadata.go
    retry.go
)

IF (OS_LINUX)
    SRCS(
        retry_linux.go
        syscheck_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        syscheck.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        syscheck_windows.go
    )
ENDIF()

END()

RECURSE(
    internal
)
