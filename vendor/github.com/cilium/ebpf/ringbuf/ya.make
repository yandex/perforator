GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.3)

SRCS(
    doc.go
)

IF (OS_LINUX)
    SRCS(
        reader.go
        ring.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        reader.go
        ring.go
    )
ENDIF()

END()

RECURSE(
    # gotest
)
