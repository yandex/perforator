GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.0.0)

SRCS(
    errors.go
    ifaces.go
    options.go
    pointer.go
)

GO_TEST_SRCS(
    dash_token_test.go
    examples_test.go
    fuzz_test.go
    options_test.go
    pointer_test.go
    testdata_test.go
)

GO_XTEST_SRCS(
    iface_example_test.go
    struct_example_test.go
)

GO_TEST_EMBED_PATTERN(testdata/*.json)

END()

RECURSE(
    gotest
    jsonname
)
