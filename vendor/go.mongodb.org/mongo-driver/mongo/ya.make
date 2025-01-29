GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    background_context.go
    batch_cursor.go
    bulk_write.go
    bulk_write_models.go
    change_stream.go
    change_stream_deployment.go
    client.go
    client_encryption.go
    collection.go
    crypt_retrievers.go
    cursor.go
    database.go
    doc.go
    errors.go
    index_options_builder.go
    index_view.go
    mongo.go
    mongocryptd.go
    results.go
    search_index_view.go
    session.go
    single_result.go
    util.go
)

GO_TEST_SRCS(
    background_context_test.go
    bson_helpers_test.go
    change_stream_test.go
    client_side_encryption_examples_test.go
    client_test.go
    collection_test.go
    cursor_test.go
    database_test.go
    errors_test.go
    mongo_test.go
    ocsp_test.go
    read_write_concern_spec_test.go
    results_test.go
    single_result_test.go
    with_transactions_test.go
)

GO_XTEST_SRCS(
    client_examples_test.go
    crud_examples_test.go
)

END()

RECURSE(
    address
    description
    # gotest
    gridfs
    integration
    options
    readconcern
    readpref
    writeconcern
)
