GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.0.0-20240424095704-91a3fc46842c)

SRCS(
    diff.go
    diff_containerd.go
    diskwriter.go
    filter.go
    followlinks.go
    fs.go
    hardlinks.go
    receive.go
    send.go
    stat.go
    tarwriter.go
    validator.go
)

GO_TEST_SRCS(
    diskwriter_test.go
    filter_test.go
    followlinks_test.go
    fs_test.go
    hardlinks_test.go
    # receive_test.go
    # stat_test.go
    validator_test.go
)

IF (OS_LINUX)
    SRCS(
        chtimes_linux.go
        diskwriter_unix.go
        diskwriter_unixnobsd.go
        followlinks_unix.go
        stat_unix.go
    )

    GO_TEST_SRCS(diskwriter_unix_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        chtimes_nolinux.go
        diskwriter_unix.go
        diskwriter_unixnobsd.go
        followlinks_unix.go
        stat_unix.go
    )

    GO_TEST_SRCS(diskwriter_unix_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        chtimes_nolinux.go
        diskwriter_windows.go
        followlinks_windows.go
        stat_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        chtimes_linux.go
        diskwriter_unix.go
        diskwriter_unixnobsd.go
        followlinks_unix.go
        stat_unix.go
    )

    GO_TEST_SRCS(diskwriter_unix_test.go)
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        chtimes_nolinux.go
        diskwriter_unix.go
        diskwriter_unixnobsd.go
        followlinks_unix.go
        stat_unix.go
    )

    GO_TEST_SRCS(diskwriter_unix_test.go)
ENDIF()

END()

RECURSE(
    gotest
    types
)
