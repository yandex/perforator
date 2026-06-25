GO_PROGRAM()

LICENSE(MIT)

VERSION(v0.17.3)

IF (OS_LINUX)
    SRCS(
        doc.go
        flags.go
        main.go
        makedep.go
        tools.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        doc.go
        flags.go
        main.go
        makedep.go
        tools.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        doc.go
        flags.go
        main.go
        makedep.go
        tools.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        doc.go
        flags.go
        main.go
        makedep.go
        tools.go
    )
ENDIF()

END()

RECURSE(
    gen
    # gotest
    internal
    test
)
