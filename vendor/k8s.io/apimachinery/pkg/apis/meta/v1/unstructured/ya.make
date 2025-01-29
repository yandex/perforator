GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    helpers.go
    unstructured.go
    unstructured_list.go
    zz_generated.deepcopy.go
)

GO_TEST_SRCS(
    helpers_test.go
    unstructured_list_test.go
)

GO_XTEST_SRCS(
    unstructured_conversion_test.go
    unstructured_test.go
)

END()

RECURSE(
    # gotest
    unstructuredscheme
)
