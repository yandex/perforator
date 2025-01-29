GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    config.go
    doc.go
    errors.go
    fs.go
    import.go
    loader.go
    match.go
    package.go
    read.go
    search.go
    source.go
    tags.go
)

GO_TEST_SRCS(
    import_test.go
    loader_test.go
    read_test.go
    search_test.go
    tags_test.go
)

END()

RECURSE(
    # gotest
)
