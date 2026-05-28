GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.1.5)

IF (OS_LINUX)
    SRCS(
        fsverity_linux.go
    )

    GO_TEST_SRCS(fsverity_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        fsverity_other.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        fsverity_other.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        fsverity_linux.go
    )

    GO_TEST_SRCS(fsverity_test.go)
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        fsverity_other.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
