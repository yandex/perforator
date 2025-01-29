# yo ignore:file
GO_LIBRARY()

USE_UTIL()

IF (CGO_ENABLED)
    PEERDIR(
        perforator/symbolizer/lib/gsym
    )

    CGO_SRCS(gsym.go)
ELSE()
    SRCS(stub.go)
ENDIF()

END()
