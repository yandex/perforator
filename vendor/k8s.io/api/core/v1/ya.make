GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    annotation_key_constants.go
    doc.go
    generated.pb.go
    lifecycle.go
    objectreference.go
    register.go
    resource.go
    taint.go
    toleration.go
    types.go
    types_swagger_doc_generated.go
    well_known_labels.go
    well_known_taints.go
    zz_generated.deepcopy.go
)

GO_TEST_SRCS(
    taint_test.go
    toleration_test.go
    types_test.go
)

END()

RECURSE(
    gotest
)
