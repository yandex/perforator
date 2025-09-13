# yo ignore:file
GO_LIBRARY()

USE_UTIL()

IF (CGO_ENABLED)
    USE_CXX()

    PEERDIR(
        perforator/symbolizer/lib/symbolize
        perforator/symbolizer/lib/stacks_sampling
    )

    CGO_SRCS(symbolize.go)
    CGO_SRCS(stacks_sampling.go)
ELSE()
    SRCS(stub.go)
    SRCS(stacks_sampling_stub.go)
ENDIF()

SRCS(
    binaries.go
    cachedbinaries.go
    errors.go
    interface.go
)

END()
