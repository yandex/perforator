GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    doc.go
    provider.go
    sso_cached_token.go
    token_provider.go
)

GO_TEST_SRCS(
    provider_test.go
    sso_cached_token_test.go
    token_provider_test.go
)

IF (OS_LINUX)
    SRCS(
        os.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        os.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        os_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
