GO_LIBRARY()

LICENSE(MIT)

VERSION(v2.23.4)

SRCS(
    counter.go
    failer.go
    focus.go
    group.go
    node.go
    ordering.go
    output_interceptor.go
    progress_report.go
    progress_reporter_manager.go
    report_entry.go
    spec.go
    spec_context.go
    suite.go
    tree.go
    writer.go
)

IF (OS_LINUX)
    SRCS(
        output_interceptor_unix.go
        progress_report_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        output_interceptor_unix.go
        progress_report_bsd.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        output_interceptor_win.go
        progress_report_win.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        output_interceptor_unix.go
        progress_report_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        output_interceptor_wasm.go
        progress_report_wasm.go
    )
ENDIF()

END()

RECURSE(
    global
    interrupt_handler
    parallel_support
    test_helpers
    testingtproxy
)
