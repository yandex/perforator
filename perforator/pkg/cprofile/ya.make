# yo ignore:file
GO_LIBRARY()

USE_UTIL()

IF (CGO_ENABLED AND NOT SANDBOXING)
    PEERDIR(
        perforator/lib/profile/c
    )

    CGO_SRCS(
        error_cgo.go
        merge_cgo.go
        profile_cgo.go
    )
ELSE()
    SRCS(
        merge_nocgo.go
        profile_nocgo.go
    )
ENDIF()

END()

RECURSE(cmd)
