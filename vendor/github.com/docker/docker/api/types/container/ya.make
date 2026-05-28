GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v28.0.4+incompatible)

SRCS(
    change_type.go
    change_types.go
    commit.go
    config.go
    container.go
    create_request.go
    create_response.go
    errors.go
    exec.go
    filesystem_change.go
    health.go
    hostconfig.go
    network_settings.go
    options.go
    port.go
    stats.go
    top_response.go
    update_response.go
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
