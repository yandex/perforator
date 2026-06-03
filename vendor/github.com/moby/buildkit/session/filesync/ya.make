GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.23.2)

SRCS(
    diffcopy.go
    filesync.go
    filesync.pb.go
    filesync_grpc.pb.go
    filesync_vtproto.pb.go
)

GO_TEST_SRCS(filesync_test.go)

IF (OS_LINUX)
    SRCS(
        diffcopy_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        diffcopy_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        diffcopy_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        diffcopy_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        diffcopy_unix.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
