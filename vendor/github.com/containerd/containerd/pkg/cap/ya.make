GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.7.25)

IF (OS_LINUX)
    SRCS(
        cap_linux.go
    )

    GO_TEST_SRCS(cap_linux_test.go)
ENDIF()

END()

RECURSE(
    gotest
)
