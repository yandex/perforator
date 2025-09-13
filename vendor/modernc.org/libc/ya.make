GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.62.1)

SRCS(
    fsync.go
    int128.go
    libc_all.go
    nodmesg.go
    probes.go
    stdatomic.go
    straceoff.go
    watch.go
)

GO_TEST_SRCS(
    # all_test.go
)

IF (OS_LINUX)
    SRCS(
        aliases.go
        atomic.go
        builtin.go
        etc_musl.go
        libc_musl.go
        mem_musl.go
        pthread_musl.go
        rtl.go
        syscall_musl.go
    )

    GO_TEST_SRCS(
        # all_musl_test.go
        unix_test.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_X86_64)
    SRCS(
        atomic64.go
        builtin64.go
        capi_linux_amd64.go
        ccgo_linux_amd64.go
        libc_musl_linux_amd64.go
    )

    GO_TEST_SRCS(malloc_test.go)
ENDIF()

IF (OS_LINUX AND ARCH_ARM64)
    SRCS(
        atomic64.go
        builtin64.go
        capi_linux_arm64.go
        ccgo_linux_arm64.go
        libc_musl_linux_arm64.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        atomic32.go
        builtin32.go
        capi_linux_arm.go
        ccgo_linux_arm.go
        libc_musl_linux_arm.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        ccgo.go
        etc.go
        ioutil_darwin.go
        libc.go
        libc64.go
        libc_darwin.go
        libc_unix.go
        libc_unix1.go
        mem.go
        printf.go
        pthread.go
        pthread_all.go
        scanf.go
        sync.go
    )

    GO_TEST_SRCS(
        # all_non_musl_test.go
        unix_test.go
    )
ENDIF()

IF (OS_DARWIN AND ARCH_X86_64)
    SRCS(
        capi_darwin_amd64.go
        libc_amd64.go
        libc_darwin_amd64.go
        musl_darwin_amd64.go
    )
ENDIF()

IF (OS_DARWIN AND ARCH_ARM64)
    SRCS(
        capi_darwin_arm64.go
        libc_arm64.go
        libc_darwin_arm64.go
        musl_darwin_arm64.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        ccgo.go
        etc.go
        libc.go
        libc64.go
        libc_windows.go
        mem.go
        printf.go
        pthread.go
        pthread_all.go
        scanf.go
        sync.go
    )

    GO_TEST_SRCS(
        # all_non_musl_test.go
    )
ENDIF()

IF (OS_WINDOWS AND ARCH_X86_64)
    SRCS(
        capi_windows_amd64.go
        libc_amd64.go
        libc_windows_amd64.go
        musl_windows_amd64.go
    )
ENDIF()

IF (OS_WINDOWS AND ARCH_ARM64)
    SRCS(
        capi_windows_arm64.go
        libc_arm64.go
        libc_windows_arm64.go
        musl_windows_arm64.go
    )
ENDIF()

END()

RECURSE(
    gotest
    honnef.co
    netinet
    sys
    uuid
)

IF (OS_LINUX AND ARCH_X86_64)
    RECURSE(
        fts
        stdlib
        unistd
        pwd
        time
        utime
        stdio
        netdb
        poll
        fcntl
        limits
        termios
        grp
        signal
        errno
        langinfo
        pthread
        wctype
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM64)
    RECURSE(
        fts
        stdlib
        unistd
        pwd
        time
        utime
        stdio
        netdb
        poll
        fcntl
        limits
        termios
        grp
        signal
        errno
        langinfo
        pthread
        wctype
    )
ENDIF()

IF (OS_DARWIN AND ARCH_X86_64)
    RECURSE(
        fts
        stdlib
        unistd
        pwd
        time
        utime
        stdio
        netdb
        poll
        fcntl
        limits
        termios
        grp
        signal
        errno
        langinfo
        pthread
        wctype
    )
ENDIF()

IF (OS_DARWIN AND ARCH_ARM64)
    RECURSE(
        fts
        stdlib
        unistd
        pwd
        time
        utime
        stdio
        netdb
        poll
        fcntl
        limits
        termios
        grp
        signal
        errno
        langinfo
        pthread
        wctype
    )
ENDIF()

IF (OS_WINDOWS)
    RECURSE(
        stdlib
        unistd
        time
        utime
        stdio
        fcntl
        limits
        signal
        errno
        pthread
        wctype
    )
ENDIF()
