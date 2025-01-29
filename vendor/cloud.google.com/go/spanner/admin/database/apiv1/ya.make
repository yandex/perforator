GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.60.0)

SRCS(
    auxiliary.go
    backup.go
    database.go
    database_admin_client.go
    doc.go
    init.go
    path_funcs.go
    version.go
)

GO_TEST_SRCS(
    backup_test.go
    database_test.go
    mock_test.go
)

GO_XTEST_SRCS(database_admin_client_example_test.go)

END()

RECURSE(
    databasepb
    gotest
)
