GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.6.1)

SRCS(
    numcpus.go
)

GO_XTEST_SRCS(
    example_test.go
    numcpus_test.go
)

IF (OS_LINUX)
    SRCS(
        numcpus_linux.go
    )

    GO_TEST_SRCS(numcpus_linux_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        numcpus_bsd.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        numcpus_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
