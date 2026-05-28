GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v28.0.4+incompatible)

SRCS(
    archive.go
    changes.go
    copy.go
    diff.go
    path.go
    time.go
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
        dev_unix.go
        diff_unix.go
        path_unix.go
        time_nonwindows.go
        xattr_supported.go
        xattr_supported_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        archive_other.go
        archive_unix.go
        changes_other.go
        changes_unix.go
        copy_unix.go
        dev_unix.go
        diff_unix.go
        path_unix.go
        time_nonwindows.go
        xattr_supported.go
        xattr_supported_unix.go
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
        time_windows.go
        xattr_unsupported.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        archive_linux.go
        archive_unix.go
        changes_linux.go
        changes_unix.go
        copy_unix.go
        dev_unix.go
        diff_unix.go
        path_unix.go
        time_nonwindows.go
        xattr_supported.go
        xattr_supported_linux.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        archive_other.go
        archive_unix.go
        changes_other.go
        changes_unix.go
        copy_unix.go
        dev_unix.go
        diff_unix.go
        path_unix.go
        time_nonwindows.go
        xattr_supported_unix.go
        xattr_unsupported.go
    )
ENDIF()

END()
