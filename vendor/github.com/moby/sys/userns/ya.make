GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.1.0)

SRCS(
    userns.go
)

IF (OS_LINUX)
    SRCS(
        userns_linux.go
    )

    GO_TEST_SRCS(userns_linux_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        userns_unsupported.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        userns_unsupported.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
