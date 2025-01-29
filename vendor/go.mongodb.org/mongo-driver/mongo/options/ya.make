GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

GO_SKIP_TESTS(TestClientOptions)

SRCS(
    aggregateoptions.go
    autoencryptionoptions.go
    bulkwriteoptions.go
    changestreamoptions.go
    clientencryptionoptions.go
    clientoptions.go
    collectionoptions.go
    countoptions.go
    createcollectionoptions.go
    datakeyoptions.go
    dboptions.go
    deleteoptions.go
    distinctoptions.go
    doc.go
    encryptoptions.go
    estimatedcountoptions.go
    findoptions.go
    gridfsoptions.go
    indexoptions.go
    insertoptions.go
    listcollectionsoptions.go
    listdatabasesoptions.go
    loggeroptions.go
    mongooptions.go
    replaceoptions.go
    rewrapdatakeyoptions.go
    runcmdoptions.go
    searchindexoptions.go
    serverapioptions.go
    sessionoptions.go
    transactionoptions.go
    updateoptions.go
)

GO_TEST_SRCS(
    # changestreamoptions_test.go
    # clientoptions_test.go
    # collation_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
