GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v3.24.2)

GO_SKIP_TESTS(
    Test_Process_Environ
    Test_Process_Long_Name
    Test_Process_Long_Name_With_Spaces
)

SRCS(
    process.go
)

GO_TEST_SRCS(process_test.go)

IF (OS_LINUX)
    SRCS(
        process_linux.go
        process_posix.go
    )

    GO_TEST_SRCS(
        process_linux_test.go
        process_posix_test.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        process_bsd.go
        process_darwin.go
        process_posix.go
    )
ENDIF()

IF (OS_DARWIN AND CGO_ENABLED)
    CGO_SRCS(process_darwin_cgo.go)
ENDIF()

IF (OS_DARWIN AND ARCH_X86_64)
    SRCS(
        process_darwin_amd64.go
    )
ENDIF()

IF (OS_DARWIN AND ARCH_ARM64)
    SRCS(
        process_darwin_arm64.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        process_windows.go
        process_windows_64bit.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
