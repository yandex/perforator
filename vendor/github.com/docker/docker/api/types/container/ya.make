GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v27.5.1+incompatible)

SRCS(
    change_type.go
    change_types.go
    config.go
    container.go
    container_top.go
    container_update.go
    create_request.go
    create_response.go
    errors.go
    exec.go
    filesystem_change.go
    hostconfig.go
    options.go
    stats.go
    wait_exit_error.go
    wait_response.go
    waitcondition.go
)

IF (OS_LINUX)
    SRCS(
        hostconfig_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        hostconfig_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        hostconfig_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        hostconfig_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        hostconfig_unix.go
    )
ENDIF()

END()
