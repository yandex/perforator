GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.76.1)

SRCS(
    auxiliary.go
    auxiliary_go123.go
    backup.go
    database.go
    database_admin_client.go
    doc.go
    helpers.go
    init.go
    path_funcs.go
    version.go
)

GO_TEST_SRCS(
    backup_test.go
    database_test.go
    mock_test.go
)

GO_XTEST_SRCS(
    database_admin_client_example_go123_test.go
    database_admin_client_example_test.go
)

END()

RECURSE(
    databasepb
    gotest
)
