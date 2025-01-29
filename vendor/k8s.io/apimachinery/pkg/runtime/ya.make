GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    allocator.go
    codec.go
    codec_check.go
    conversion.go
    converter.go
    doc.go
    embedded.go
    error.go
    extension.go
    generated.pb.go
    helper.go
    interfaces.go
    mapper.go
    negotiate.go
    register.go
    scheme.go
    scheme_builder.go
    swagger_doc_generator.go
    types.go
    types_proto.go
    zz_generated.deepcopy.go
)

GO_TEST_SRCS(
    allocator_test.go
    local_scheme_test.go
    mapper_test.go
    swagger_doc_generator_test.go
)

GO_XTEST_SRCS(
    codec_test.go
    converter_test.go
    embedded_test.go
    extension_test.go
    scheme_test.go
)

END()

RECURSE(
    #    gotest
    schema
    serializer
    testing
)
