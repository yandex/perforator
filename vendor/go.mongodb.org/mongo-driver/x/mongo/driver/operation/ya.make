GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    abort_transaction.go
    aggregate.go
    command.go
    commit_transaction.go
    count.go
    create.go
    create_indexes.go
    create_search_indexes.go
    delete.go
    distinct.go
    doc.go
    drop_collection.go
    drop_database.go
    drop_indexes.go
    drop_search_index.go
    end_sessions.go
    errors.go
    find.go
    find_and_modify.go
    hello.go
    insert.go
    listDatabases.go
    list_collections.go
    list_indexes.go
    update.go
    update_search_index.go
)

GO_TEST_SRCS(hello_test.go)

END()

RECURSE(
    gotest
)
