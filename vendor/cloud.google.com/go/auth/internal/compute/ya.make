GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.2)

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

IF (OS_ANDROID)
    SRCS(
        manufacturer_linux.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        manufacturer.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
