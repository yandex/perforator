GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.36.0)

SRCS(
    auto.go
    builtin.go
    config.go
    container.go
    doc.go
    env.go
    host_id.go
    os.go
    process.go
    resource.go
)

GO_TEST_SRCS(
    container_test.go
    # env_test.go
    export_test.go
    host_id_test.go
    resource_experimental_test.go
)

GO_XTEST_SRCS(
    auto_test.go
    benchmark_test.go
    builtin_test.go
    example_test.go
    host_id_export_test.go
    os_test.go
    process_test.go
    resource_test.go
)

IF (OS_LINUX)
    SRCS(
        host_id_linux.go
        host_id_readfile.go
        os_release_unix.go
        os_unix.go
    )

    GO_TEST_SRCS(
        export_common_unix_test.go
        export_unix_test.go
        host_id_readfile_test.go
    )

    GO_XTEST_SRCS(
        os_release_unix_test.go
        os_unix_test.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        host_id_darwin.go
        host_id_exec.go
        os_release_darwin.go
        os_unix.go
    )

    GO_TEST_SRCS(
        export_common_unix_test.go
        export_os_release_darwin_test.go
    )

    GO_XTEST_SRCS(
        os_release_darwin_test.go
        os_unix_test.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        host_id_windows.go
        os_windows.go
    )

    GO_TEST_SRCS(
        export_windows_test.go
        host_id_windows_test.go
    )

    GO_XTEST_SRCS(os_windows_test.go)
ENDIF()

END()

RECURSE(
    gotest
)
