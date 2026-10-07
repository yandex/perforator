GO_LIBRARY()

SRCS(
    common.go
    fetch.go
    list.go
    microscope.go
    profile.go
    root.go
    sink.go
    symbolize.go
    symbolize_batch.go
)

GO_TEST_SRCS(symbolize_batch_test.go)

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

RECURSE(
    gotest
)
