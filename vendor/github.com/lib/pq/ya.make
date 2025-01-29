GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.10.9)

SRCS(
    array.go
    buf.go
    conn.go
    conn_go115.go
    conn_go18.go
    connector.go
    copy.go
    doc.go
    encode.go
    error.go
    krb.go
    notice.go
    notify.go
    rows.go
    ssl.go
    url.go
    uuid.go
)

GO_TEST_SRCS(
    array_test.go
    bench_test.go
    buf_test.go
    conn_test.go
    connector_test.go
    copy_test.go
    encode_test.go
    go18_test.go
    go19_test.go
    issues_test.go
    notice_test.go
    notify_test.go
    rows_test.go
    ssl_test.go
    url_test.go
    uuid_test.go
)

GO_XTEST_SRCS(
    connector_example_test.go
    notice_example_test.go
)

IF (OS_LINUX)
    SRCS(
        ssl_permissions.go
        user_posix.go
    )

    GO_TEST_SRCS(ssl_permissions_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        ssl_permissions.go
        user_posix.go
    )

    GO_TEST_SRCS(ssl_permissions_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        ssl_windows.go
        user_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
    oid
    scram
)
