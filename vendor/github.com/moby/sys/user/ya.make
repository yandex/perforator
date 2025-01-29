GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.1.0)

SRCS(
    user.go
)

GO_TEST_SRCS(user_test.go)

IF (OS_LINUX)
    SRCS(
        lookup_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        lookup_unix.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
