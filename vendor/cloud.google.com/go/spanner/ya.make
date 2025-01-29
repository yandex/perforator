GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.60.0)

GO_SKIP_TESTS(
    TestTakeFromIdleWriteListChecked
    TestClient_CallOptions
    # flaky
)

SRCS(
    batch.go
    client.go
    doc.go
    errors.go
    errors113.go
    key.go
    mutation.go
    ot_metrics.go
    pdml.go
    protoutils.go
    read.go
    retry.go
    row.go
    session.go
    sessionclient.go
    statement.go
    stats.go
    timestampbound.go
    transaction.go
    value.go
)

GO_TEST_SRCS(
    # batch_test.go
    client_benchmarks_test.go
    client_test.go
    cmp_test.go
    errors_test.go
    # integration_test.go
    key_test.go
    mocks_test.go
    mutation_test.go
    # oc_test.go
    pdml_test.go
    read_test.go
    retry_test.go
    # row_test.go
    session_test.go
    sessionclient_test.go
    statement_test.go
    timestampbound_test.go
    transaction_test.go
    value_benchmarks_test.go
    value_test.go
)

GO_XTEST_SRCS(examples_test.go)

END()

RECURSE(
    admin
    apiv1
    executor
    gotest
    internal
    spannertest
    spansql
    test
)
