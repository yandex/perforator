GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.34.1)

SRCS(
    doc.go
    fcntl.go
    mutex.go
    nodmesg.go
    sqlite.go
    sqlite_go18.go
)

GO_TEST_SRCS(
    # all_test.go
    fcntl_test.go
    func_test.go
    # null_test.go
    # sqlite_go18_test.go
)

IF (OS_LINUX)
    SRCS(
        rulimit.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_X86_64)
    SRCS(
        bind_blob_musl.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM64)
    SRCS(
        bind_blob_musl.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        bind_blob.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        bind_blob.go
        rulimit.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        bind_blob.go
        norlimit.go
    )
ENDIF()

GO_TEST_EMBED_PATTERN(embed.db)

GO_TEST_EMBED_PATTERN(embed2.db)

END()

RECURSE(
    gotest
    lib
    vfs
)
