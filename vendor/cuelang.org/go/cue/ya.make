GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

GO_SKIP_TESTS(
    TestBuild
    TestMarshalling
    TestMarshalMultiPackage
    TestReference
    TestReferencePath
    TestValidate
    TestValueType
)

SRCS(
    attribute.go
    build.go
    builtin.go
    builtinutil.go
    context.go
    cue.go
    decode.go
    errors.go
    format.go
    instance.go
    marshal.go
    op.go
    path.go
    query.go
    types.go
)

GO_TEST_SRCS(
    attribute_test.go
    decode_test.go
    marshal_test.go
    path_test.go
    resolve_test.go
    types_test.go
)

GO_XTEST_SRCS(
    # build_test.go
    # builtin_test.go
    # context_test.go
    # examplecompile_test.go
    # examples_test.go
    # format_test.go
    # query_test.go
    # syntax_test.go
)

END()

RECURSE(
    ast
    build
    cuecontext
    errors
    format
    gotest
    literal
    load
    parser
    scanner
    token
)
