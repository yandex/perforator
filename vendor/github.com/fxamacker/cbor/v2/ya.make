GO_LIBRARY()

LICENSE(MIT)

VERSION(v2.9.0)

SRCS(
    bytestring.go
    cache.go
    common.go
    decode.go
    diagnose.go
    doc.go
    encode.go
    encode_map.go
    omitzero_go124.go
    simplevalue.go
    stream.go
    structfields.go
    tag.go
    valid.go
)

GO_TEST_SRCS(
    bench_test.go
    bytestring_test.go
    decode_test.go
    diagnose_test.go
    encode_test.go
    simplevalue_test.go
    stream_test.go
    tag_test.go
    valid_test.go
)

GO_XTEST_SRCS(
    example_embedded_json_tag_for_cbor_test.go
    example_test.go
    example_transcoding_test.go
    json_test.go
)

END()

RECURSE(
    gotest
)
