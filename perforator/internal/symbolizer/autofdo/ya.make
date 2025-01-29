# yo ignore:file
GO_LIBRARY()

USE_UTIL()

IF (CGO_ENABLED AND NOT SANDBOXING)
    PEERDIR(
        perforator/symbolizer/lib/autofdo
    )

    CGO_SRCS(autofdo.go)
ELSE()
    SRCS(stub.go)
ENDIF()

END()
