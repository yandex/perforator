GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.3)

SRCS(
    doc.go
)

IF (OS_LINUX)
    SRCS(
        compile.go
        output.go
        target.go
        types.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        compile.go
        output.go
        target.go
        types.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        compile.go
        output.go
        target.go
        types.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        compile.go
        output.go
        target.go
        types.go
    )
ENDIF()

GO_EMBED_PATTERN(output.tpl)

END()
