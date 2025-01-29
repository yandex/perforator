GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.5.0)

SRCS(
    ascii.go
    doc.go
    proxy.go
    term.go
)

GO_TEST_SRCS(
    ascii_test.go
    proxy_test.go
)

IF (OS_LINUX)
    SRCS(
        term_unix.go
        termios_nonbsd.go
        termios_unix.go
    )

    GO_TEST_SRCS(term_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        term_unix.go
        termios_bsd.go
        termios_unix.go
    )

    GO_TEST_SRCS(term_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        term_windows.go
        termios_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
    windows
)
