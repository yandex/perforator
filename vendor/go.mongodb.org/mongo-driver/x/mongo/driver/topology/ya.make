GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

GO_SKIP_TESTS(
    TestCMAPSpec
    TestPool
)

SRCS(
    cancellation_listener.go
    connection.go
    connection_legacy.go
    connection_options.go
    diff.go
    errors.go
    fsm.go
    pool.go
    pool_generation_counter.go
    rtt_monitor.go
    server.go
    server_options.go
    tls_connection_source_1_17.go
    topology.go
    topology_options.go
)

GO_TEST_SRCS(
    CMAP_spec_test.go
    cmap_prose_test.go
    connection_errors_test.go
    connection_test.go
    diff_test.go
    fsm_test.go
    polling_srv_records_test.go
    pool_test.go
    rtt_monitor_test.go
    sdam_spec_test.go
    # server_rtt_test.go
    # server_test.go
    topology_errors_test.go
    topology_options_test.go
    topology_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
