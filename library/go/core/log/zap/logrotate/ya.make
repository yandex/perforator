GO_LIBRARY()

SRCS(
    error.go
)

IF (OS_LINUX OR OS_ANDROID)
    SRCS(
        sink.go
    )

    GO_TEST_SRCS(
        benchmark_test.go
        integration_test.go
        signal_test.go
        sink_test.go
    )

    GO_XTEST_SRCS(example_sink_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        sink.go
    )

    GO_TEST_SRCS(
        benchmark_test.go
        integration_test.go
        signal_test.go
        sink_test.go
    )

    GO_XTEST_SRCS(example_sink_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        sink_stub.go
    )
ENDIF()

END()

IF (OS_DARWIN OR OS_FREEBSD OR OS_LINUX OR OS_ANDROID)
    RECURSE_FOR_TESTS(
        gotest
    )
ENDIF()
