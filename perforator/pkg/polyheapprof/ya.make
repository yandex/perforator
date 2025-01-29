# yo ignore:file
GO_LIBRARY()

USE_UTIL()

IF (NOT OPENSOURCE AND CGO_ENABLED)
    PEERDIR(
        yt/yt/library/ytprof
        library/cpp/yt/backtrace/absl_unwinder
    )

    CGO_SRCS(heap_cgo.go)
    SRCS(heap.cpp)
ELSE()
    SRCS(heap_stub.go)
ENDIF()

SRCS(
    heap.go
)

END()
