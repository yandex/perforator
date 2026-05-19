GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.7.29)

SRCS(
    authorizer.go
    converter.go
    errcode.go
    errdesc.go
    fetcher.go
    handler.go
    httpreadseeker.go
    pusher.go
    registry.go
    resolver.go
    scope.go
    status.go
)

GO_TEST_SRCS(
    fetcher_test.go
    handler_test.go
    pusher_test.go
    registry_test.go
    resolver_test.go
    scope_test.go
)

IF (OS_LINUX)
    SRCS(
        resolver_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        resolver_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        resolver_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        resolver_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        resolver_unix.go
    )
ENDIF()

END()

RECURSE(
    auth
    gotest
    schema1
)
