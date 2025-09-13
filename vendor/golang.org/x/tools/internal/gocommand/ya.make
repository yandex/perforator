GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.35.1-0.20250728180453-01a3475a31bc)

SRCS(
    invoke.go
    vendor.go
    version.go
)

IF (OS_LINUX)
    SRCS(
        invoke_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        invoke_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        invoke_notunix.go
    )
ENDIF()

END()
