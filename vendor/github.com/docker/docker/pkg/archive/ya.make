GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v25.0.5+incompatible)

SRCS(
    archive.go
    changes.go
    copy.go
    diff.go
    path.go
    whiteouts.go
    wrap.go
)

IF (OS_LINUX)
    SRCS(
        archive_linux.go
        archive_unix.go
        changes_linux.go
        changes_unix.go
        copy_unix.go
        diff_unix.go
        path_unix.go
        time_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        archive_other.go
        archive_unix.go
        changes_other.go
        changes_unix.go
        copy_unix.go
        diff_unix.go
        path_unix.go
        time_unsupported.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        archive_other.go
        archive_windows.go
        changes_other.go
        changes_windows.go
        copy_windows.go
        diff_windows.go
        path_windows.go
        time_unsupported.go
    )
ENDIF()

END()
