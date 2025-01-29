GO_LIBRARY()

LICENSE(MIT)

VERSION(v5.7.1)

SRCS(
    batch_results.go
    conn.go
    doc.go
    pool.go
    rows.go
    stat.go
    tracer.go
    tx.go
)

GO_XTEST_SRCS(
    bench_test.go
    common_test.go
    # conn_test.go
    # pool_test.go
    tracer_test.go
    # tx_test.go
)

END()

RECURSE(
    gotest
)
