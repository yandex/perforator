# yo ignore:file
GO_LIBRARY()

USE_UTIL()

IF (CGO_ENABLED)
    USE_CXX()

    PEERDIR(
        perforator/symbolizer/lib/symbolize
    )

    CGO_SRCS(symbolize.go)
ELSE()
    SRCS(stub.go)
ENDIF()

SRCS(
    binaries.go
    cachedbinaries.go
    errors.go
)

END()
