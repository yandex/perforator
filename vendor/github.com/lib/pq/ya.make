GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.12.0)

SRCS(
    array.go
    as_go126.go
    buf.go
    conn.go
    conn_go18.go
    connector.go
    copy.go
    deprecated.go
    doc.go
    encode.go
    error.go
    krb.go
    notice.go
    notify.go
    quote.go
    rows.go
    ssl.go
    stmt.go
)

GO_TEST_SRCS(
    array_test.go
    bench_test.go
    conn_test.go
    connector_test.go
    copy_test.go
    deprecated_test.go
    encode_test.go
    error_test.go
    fuzz_test.go
    helper_test.go
    issues_test.go
    notice_test.go
    notify_test.go
    quote_test.go
    rows_test.go
    ssl_test.go
)

GO_XTEST_SRCS(
    example126_test.go
    example_test.go
)

END()

RECURSE(
    gotest
    internal
    oid
    pqerror
    scram
)
