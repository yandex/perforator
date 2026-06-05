GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.2.0)

SRCS(
    authorizer.go
    converter.go
    errcode.go
    errdesc.go
    fetcher.go
    handler.go
    httpreadseeker.go
    pusher.go
    referrers.go
    registry.go
    resolver.go
    scope.go
    status.go
)

GO_TEST_SRCS(
    converter_fuzz_test.go
    fetcher_fuzz_test.go
    fetcher_test.go
    handler_test.go
    pusher_test.go
    referrers_test.go
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
)
