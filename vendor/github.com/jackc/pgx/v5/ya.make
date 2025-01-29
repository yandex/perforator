GO_LIBRARY()

LICENSE(MIT)

VERSION(v5.7.1)

SRCS(
    batch.go
    conn.go
    copy_from.go
    derived_types.go
    doc.go
    extended_query_builder.go
    large_objects.go
    named_args.go
    rows.go
    tracer.go
    tx.go
    values.go
)

GO_TEST_SRCS(
    # conn_internal_test.go
    # large_objects_private_test.go
)

GO_XTEST_SRCS(
    # batch_test.go
    # bench_test.go
    # conn_test.go
    # copy_from_test.go
    # derived_types_test.go
    # helper_test.go
    # large_objects_test.go
    # named_args_test.go
    # pgbouncer_test.go
    # pgx_test.go
    # pipeline_test.go
    # query_test.go
    # rows_test.go
    # tracer_test.go
    # tx_test.go
    # values_test.go
)

END()

RECURSE(
    examples
    gotest
    internal
    log
    multitracer
    pgconn
    pgproto3
    pgtype
    pgxpool
    pgxtest
    stdlib
    testsetup
    tracelog
)
