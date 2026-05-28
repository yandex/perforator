GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.1.5)

IF (OS_LINUX)
    SRCS(
        kernel_linux.go
    )

    GO_TEST_SRCS(kernel_linux_test.go)
ENDIF()

IF (OS_ANDROID)
    SRCS(
        kernel_linux.go
    )

    GO_TEST_SRCS(kernel_linux_test.go)
ENDIF()

END()

RECURSE(
    gotest
)
