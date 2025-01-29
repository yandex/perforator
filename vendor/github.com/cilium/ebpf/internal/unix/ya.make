GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.1)

SRCS(
    doc.go
)

IF (OS_LINUX)
    SRCS(
        types_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        types_other.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        types_other.go
    )
ENDIF()

END()
