GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.12.2)

GO_SKIP_TESTS(TestAccessTokenConnectorFailsToConnectIfNoAccessToken)

SRCS(
    accesstokenconnector.go
    buf.go
    bulkcopy.go
    bulkcopy_sql.go
    convert.go
    doc.go
    error.go
    fedauth.go
    log.go
    mssql.go
    mssql_go110.go
    mssql_go118.go
    mssql_go19.go
    net.go
    rpc.go
    tds.go
    token.go
    token_string.go
    tran.go
    tvp_go19.go
    types.go
    uniqueidentifier.go
)

GO_TEST_SRCS(
    accesstokenconnector_test.go
    bad_server_test.go
    buf_test.go
    bulkcopy_test.go
    error_test.go
    log_go113_test.go
    log_test.go
    messages_benchmark_test.go
    mssql_perf_test.go
    mssql_test.go
    net_go116_test.go
    queries_go110_test.go
    queries_go19_test.go
    queries_test.go
    tds_go110_test.go
    tds_login_test.go
    tds_test.go
    token_test.go
    tvp_go19_db_test.go
    tvp_go19_test.go
    types_test.go
    uniqueidentifier_test.go
)

GO_XTEST_SRCS(
    bulkimport_example_test.go
    datetimeoffset_example_test.go
    error_example_test.go
    lastinsertid_example_test.go
    messages_example_test.go
    newconnector_example_test.go
    tvp_example_test.go
)

IF (OS_LINUX)
    SRCS(
        ntlm.go
    )

    GO_TEST_SRCS(ntlm_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        ntlm.go
    )

    GO_TEST_SRCS(ntlm_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        sspi_windows.go
    )
ENDIF()

END()

RECURSE(
    azuread
    batch
    examples
    gotest
    internal
    msdsn
)
