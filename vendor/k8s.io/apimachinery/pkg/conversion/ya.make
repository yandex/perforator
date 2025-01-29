GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    converter.go
    deep_equal.go
    doc.go
    helper.go
)

GO_TEST_SRCS(
    converter_test.go
    helper_test.go
)

END()

RECURSE(
    gotest
    queryparams
)
