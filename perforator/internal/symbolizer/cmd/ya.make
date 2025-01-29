GO_LIBRARY()

SRCS(
    fetch.go
    list.go
    microscope.go
    root.go
    sink.go
    symbolize.go
)

IF (OS_LINUX)
    SRCS(
        record_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        record_other.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        record_other.go
    )
ENDIF()

END()
