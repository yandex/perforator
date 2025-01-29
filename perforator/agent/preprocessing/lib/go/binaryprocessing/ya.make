# yo ignore:file

GO_LIBRARY()

USE_UTIL()

PEERDIR(
    perforator/agent/preprocessing/proto/unwind
)

IF (CGO_ENABLED)
    SRCS(
        CGO_EXPORT
        ehframe.cpp
    )

    PEERDIR(
        perforator/agent/preprocessing/lib
    )

    CGO_SRCS(
        ehframe.go
    )
ELSE()
    SRCS(stub.go)
ENDIF()

SRCS(
    iface.go
    counting.go
)

END()
