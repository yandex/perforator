GO_LIBRARY()

LICENSE(MPL-2.0)

VERSION(v1.7.1)

SRCS(
    assert.go
    invocations.go
    must.go
    scripts.go
    settings.go
)

GO_TEST_SRCS(
    assert_test.go
    invocations_test.go
    must_test.go
    scripts_test.go
    settings_test.go
)

IF (OS_LINUX)
    SRCS(
        fs_default.go
    )

    GO_TEST_SRCS(examples_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        fs_default.go
    )

    GO_TEST_SRCS(examples_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        fs_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        fs_default.go
    )

    GO_TEST_SRCS(examples_test.go)
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        fs_default.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
