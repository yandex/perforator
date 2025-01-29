GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v25.0.5+incompatible)

SRCS(
    idtools.go
)

IF (OS_LINUX)
    SRCS(
        idtools_unix.go
        usergroupadd_linux.go
        utils_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        idtools_unix.go
        usergroupadd_unsupported.go
        utils_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        idtools_windows.go
        usergroupadd_unsupported.go
    )
ENDIF()

END()
