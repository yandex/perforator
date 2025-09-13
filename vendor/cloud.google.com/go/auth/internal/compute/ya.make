GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.15.0)

SRCS(
    compute.go
)

GO_TEST_SRCS(compute_test.go)

IF (OS_LINUX)
    SRCS(
        manufacturer_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        manufacturer.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        manufacturer_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
