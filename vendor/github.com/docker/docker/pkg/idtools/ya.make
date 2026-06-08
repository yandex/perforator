GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v28.2.2+incompatible)

SRCS(
    idtools.go
)

IF (OS_WINDOWS)
    SRCS(
        idtools_windows.go
    )
ENDIF()

END()
