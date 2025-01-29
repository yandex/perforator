GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    controller_ref.go
    conversion.go
    deepcopy.go
    doc.go
    duration.go
    generated.pb.go
    group_version.go
    helpers.go
    labels.go
    meta.go
    micro_time.go
    micro_time_fuzz.go
    micro_time_proto.go
    register.go
    time.go
    time_fuzz.go
    time_proto.go
    types.go
    types_swagger_doc_generated.go
    watch.go
    zz_generated.conversion.go
    zz_generated.deepcopy.go
    zz_generated.defaults.go
)

GO_TEST_SRCS(
    controller_ref_test.go
    duration_test.go
    group_version_test.go
    helpers_test.go
    labels_test.go
    micro_time_test.go
    options_test.go
    time_test.go
    types_test.go
)

GO_XTEST_SRCS(conversion_test.go)

END()

RECURSE(
    #    gotest
    unstructured
    validation
)
