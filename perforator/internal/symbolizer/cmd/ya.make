GO_LIBRARY()

SRCS(
    cluster_top.go
    cluster_top_query.go
    common.go
    fetch.go
    list.go
    microscope.go
    profile.go
    profile_merge.go
    root.go
    sink.go
    symbolize.go
    symbolize_batch.go
    symbolize_wire.go
)

GO_TEST_SRCS(
    symbolize_batch_test.go
    profile_merge_test.go
)

IF (CGO_ENABLED)
    GO_TEST_SRCS(
        symbolize_wire_cli_test.go
    )
ENDIF()

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
