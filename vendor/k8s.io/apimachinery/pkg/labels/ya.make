GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    doc.go
    labels.go
    selector.go
    zz_generated.deepcopy.go
)

GO_TEST_SRCS(
    labels_test.go
    selector_test.go
)

END()

RECURSE(
    gotest
)
