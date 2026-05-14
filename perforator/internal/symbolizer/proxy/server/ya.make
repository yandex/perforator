GO_LIBRARY()

SRCS(
    access.go
    banned_users.go
    config.go
    debuginfod.go
    llvm_tools.go
    merge.go
    methods.go
    microscope.go
    render.go
    server.go
    services.go
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
