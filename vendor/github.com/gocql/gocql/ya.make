GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.6.0)

SRCS(
    address_translators.go
    cluster.go
    compressor.go
    conn.go
    connectionpool.go
    control.go
    cqltypes.go
    debug_off.go
    dial.go
    doc.go
    errors.go
    events.go
    filters.go
    frame.go
    helpers.go
    host_source.go
    logger.go
    marshal.go
    metadata.go
    policies.go
    prepared_cache.go
    query_executor.go
    ring.go
    session.go
    token.go
    topology.go
    uuid.go
    version.go
)

GO_TEST_SRCS(
    address_translators_test.go
    cluster_test.go
    common_test.go
    compressor_test.go
    control_test.go
    events_test.go
    filters_test.go
    frame_test.go
    framer_bench_test.go
    helpers_test.go
    metadata_test.go
    policies_test.go
    ring_test.go
    session_connect_test.go
    token_test.go
    topology_test.go
)

GO_XTEST_SRCS(
    example_batch_test.go
    example_dynamic_columns_test.go
    example_lwt_batch_test.go
    example_lwt_test.go
    example_marshaler_test.go
    example_nulls_test.go
    example_paging_test.go
    example_set_test.go
    example_test.go
    example_udt_map_test.go
    example_udt_marshaler_test.go
    example_udt_struct_test.go
    example_udt_unmarshaler_test.go
)

END()

RECURSE(
    gotest
    internal
)
