GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.13.2)

SRCS(
    build.go
    client.go
    diskusage.go
    exporters.go
    filter.go
    graph.go
    info.go
    prune.go
    solve.go
    status.go
    workers.go
)

GO_TEST_SRCS(
    build_test.go
    client_test.go
    mergediff_test.go
    validation_test.go
)

IF (OS_LINUX)
    SRCS(
        client_unix.go
    )

    GO_TEST_SRCS(mergediff_linux_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        client_unix.go
    )

    GO_TEST_SRCS(mergediff_nolinux_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        client_windows.go
    )

    GO_TEST_SRCS(mergediff_nolinux_test.go)
ENDIF()

IF (OS_ANDROID)
    SRCS(
        client_unix.go
    )

    GO_TEST_SRCS(mergediff_linux_test.go)
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        client_unix.go
    )

    GO_TEST_SRCS(mergediff_nolinux_test.go)
ENDIF()

END()

RECURSE(
    buildid
    connhelper
    gotest
    llb
    ociindex
)
