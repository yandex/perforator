GO_LIBRARY()

SRCS(
    errors.go
)

IF (OS_LINUX)
    SRCS(
        reader_linux.go
    )
    GO_TEST_SRCS(reader_linux_test.go)
ELSE()
    SRCS(
        reader_stub.go
    )
ENDIF()

END()

RECURSE_FOR_TESTS(
    gotest
)
