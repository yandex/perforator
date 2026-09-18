# yo ignore:file
GO_LIBRARY()

USE_UTIL()

IF (CGO_ENABLED)
    PEERDIR(
        perforator/lib/profile/c
    )

    CGO_SRCS(
        error_cgo.go
        flamegraph_cgo.go
        merge_cgo.go
        profile_cgo.go
    )

    SRCS(
        bytes_cgo.go
    )

    GO_TEST_SRCS(
        bytes_cgo_test.go
    )
ELSE()
    SRCS(
        flamegraph_nocgo.go
        merge_nocgo.go
        profile_nocgo.go
    )
ENDIF()

END()

RECURSE_FOR_TESTS(
    gotest
)
