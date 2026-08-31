GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.140.0)

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

IF (OS_ANDROID)
    SRCS(
        klog_file_others.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        klog_file_others.go
    )
ENDIF()

END()

RECURSE(
    integration_tests
    internal
    klogr
    ktesting
    test
    textlogger
)
