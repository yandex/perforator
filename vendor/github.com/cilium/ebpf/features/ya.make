GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.3)

SRCS(
    doc.go
)

IF (OS_LINUX)
    SRCS(
        map.go
        misc.go
        prog.go
        version.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        map.go
        misc.go
        prog.go
        version.go
    )
ENDIF()

END()

RECURSE(
    # gotest
)
