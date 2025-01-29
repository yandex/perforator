GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v25.0.5+incompatible)

SRCS(
    chtimes.go
    errors.go
    filesys.go
    image_os_deprecated.go
    xattrs.go
)

IF (OS_LINUX)
    SRCS(
        chtimes_nowindows.go
        filesys_unix.go
        lstat_unix.go
        mknod.go
        mknod_unix.go
        stat_linux.go
        stat_unix.go
        utimes_unix.go
        xattrs_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        chtimes_nowindows.go
        filesys_unix.go
        lstat_unix.go
        mknod.go
        mknod_unix.go
        stat_darwin.go
        stat_unix.go
        utimes_unsupported.go
        xattrs_unsupported.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        args_windows.go
        chtimes_windows.go
        filesys_windows.go
        init_windows.go
        lstat_windows.go
        stat_windows.go
        utimes_unsupported.go
        xattrs_unsupported.go
    )
ENDIF()

END()
