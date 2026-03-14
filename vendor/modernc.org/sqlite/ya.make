GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.40.1)

SRCS(
    convert.go
    doc.go
    fcntl.go
    mutex.go
    nodmesg.go
    pre_update_hook.go
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

GO_XTEST_SRCS(pre_update_hook_test.go)

IF (OS_LINUX)
    SRCS(
        rulimit.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        rulimit.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        norlimit.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        rulimit.go
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
