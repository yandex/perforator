GO_LIBRARY()

LICENSE(MIT)

VERSION(v4.18.3)

SRCS(
    batch.go
    conn.go
    copy_from.go
    doc.go
    extended_query_builder.go
    go_stdlib.go
    large_objects.go
    logger.go
    messages.go
    rows.go
    tx.go
    values.go
)

GO_XTEST_SRCS(
    # batch_test.go
    # bench_test.go
    # conn_test.go
    # copy_from_test.go
    # example_custom_type_test.go
    # example_json_test.go
    helper_test.go
    # large_objects_test.go
    # pgbouncer_test.go
    # query_test.go
    # tx_test.go
    # values_test.go
)

END()

RECURSE(
    examples
    gotest
    internal
    log
    pgxpool
    stdlib
)
