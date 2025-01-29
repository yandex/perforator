GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.29.8)

SRCS(
    doc.go
    mutex.go
    nodmesg.go
    sqlite.go
    sqlite_go18.go
)

GO_TEST_SRCS(
    # all_test.go
    # null_test.go
    # sqlite_go18_test.go
)

IF (ARCH_ARM64)
    SRCS(
        bind_blob.go
    )
ENDIF()

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

IF (OS_DARWIN)
    SRCS(
        rulimit.go
    )
ENDIF()

IF (OS_DARWIN AND ARCH_X86_64)
    SRCS(
        bind_blob.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        norlimit.go
    )
ENDIF()

IF (OS_WINDOWS AND ARCH_X86_64)
    SRCS(
        bind_blob.go
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
