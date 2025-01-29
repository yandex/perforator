GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v25.0.5+incompatible)

SRCS(
    change_type.go
    change_types.go
    config.go
    container_top.go
    container_update.go
    create_response.go
    errors.go
    filesystem_change.go
    hostconfig.go
    options.go
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

END()
