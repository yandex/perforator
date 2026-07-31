GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.12.0)

SRCS(
    path.go
    pqutil.go
)

IF (OS_LINUX)
    SRCS(
        perm.go
        user_posix.go
    )

    GO_XTEST_SRCS(perm_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        perm.go
        user_posix.go
    )

    GO_XTEST_SRCS(perm_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        perm_unsupported.go
        user_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        perm.go
        user_other.go
    )

    GO_XTEST_SRCS(perm_test.go)
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        perm.go
        user_other.go
    )

    GO_XTEST_SRCS(perm_test.go)
ENDIF()

END()

RECURSE(
    gotest
)
