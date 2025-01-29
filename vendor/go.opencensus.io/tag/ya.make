GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    context.go
    doc.go
    key.go
    map.go
    map_codec.go
    metadata.go
    profile_19.go
    validate.go
)

GO_TEST_SRCS(
    map_codec_test.go
    map_test.go
    validate_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
