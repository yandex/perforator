GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.46.1)

SRCS(
    backup.go
    conn.go
    convert.go
    doc.go
    driver.go
    error.go
    fcntl.go
    mutex.go
    nodmesg.go
    pre_update_hook.go
    result.go
    rows.go
    sqlite.go
    stmt.go
    tx.go
    vtab.go
)

GO_TEST_SRCS(
    # all_test.go
    fcntl_test.go
    func_test.go
    module_test.go
    # null_test.go
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
    vtab
)
