GO_LIBRARY(sqlite3)

LICENSE(MIT)

VERSION(v1.14.24)

NO_COMPILER_WARNINGS()

PEERDIR(contrib/libs/sqlite3)

ADDINCL(
    contrib/libs/sqlite3
    GLOBAL
    vendor/github.com/mattn/go-sqlite3
)

CFLAGS(-DUSE_LIBSQLITE3)

SRCS(
    convert.go
    doc.go
    sqlite3_func_crypt.go
    sqlite3_go18.go
    sqlite3_opt_preupdate.go
    sqlite3_opt_preupdate_omit.go
)

GO_TEST_SRCS(
    backup_test.go
    callback_test.go
    error_test.go
    sqlite3_func_crypt_test.go
    sqlite3_go18_test.go
    sqlite3_load_extension_test.go
    sqlite3_opt_fts3_test.go
    sqlite3_opt_serialize_test.go
    sqlite3_test.go
)

IF (CGO_ENABLED)
    CGO_SRCS(
        backup.go
        callback.go
        error.go
        sqlite3.go
        sqlite3_context.go
        sqlite3_load_extension.go
        sqlite3_opt_serialize.go
        sqlite3_opt_userauth_omit.go
        sqlite3_type.go
    )
ENDIF()

IF (OS_LINUX AND CGO_ENABLED)
    CGO_SRCS(sqlite3_other.go)
ENDIF()

IF (OS_DARWIN AND CGO_ENABLED)
    CGO_SRCS(sqlite3_other.go)
ENDIF()

IF (OS_WINDOWS AND CGO_ENABLED)
    CGO_SRCS(
        sqlite3_usleep_windows.go
        sqlite3_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
