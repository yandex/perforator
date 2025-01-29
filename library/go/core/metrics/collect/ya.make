GO_LIBRARY()

SRCS(
    collect.go
)

IF (NOT OPENSOURCE)
    SRCS(
        system.go # depends on library/go/core/buildinfo
    )
ENDIF()

END()

RECURSE(
    policy
)
