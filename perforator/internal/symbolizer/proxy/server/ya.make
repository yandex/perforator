GO_LIBRARY()

SRCS(
    banned_users.go
    config.go
    llvm_tools.go
    microscope.go
    render.go
    server.go
    tasks.go
)

IF (OPENSOURCE)
    SRCS(
        auth.go
    )
ELSE()
    SRCS(
        auth_yandex.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
