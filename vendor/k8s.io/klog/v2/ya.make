GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.120.1)

SRCS(
    contextual.go
    contextual_slog.go
    exit.go
    format.go
    imports.go
    k8s_references.go
    k8s_references_slog.go
    klog.go
    klog_file.go
    klogr.go
    klogr_slog.go
    safeptr.go
)

GO_TEST_SRCS(
    # klog_test.go
    # klog_wrappers_test.go
)

GO_XTEST_SRCS(
    contextual_slog_example_test.go
    contextual_test.go
    exit_test.go
    format_test.go
    klogr_helper_test.go
    klogr_slog_test.go
    klogr_test.go
    output_test.go
    safeptr_test.go
)

IF (OS_LINUX)
    SRCS(
        klog_file_others.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        klog_file_others.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        klog_file_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
    integration_tests
    internal
    klogr
    ktesting
    test
    textlogger
)
