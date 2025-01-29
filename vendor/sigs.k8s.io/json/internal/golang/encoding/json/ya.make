GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v0.0.0-20221116044647-bc3834ca7abd)

SRCS(
    decode.go
    encode.go
    fold.go
    indent.go
    kubernetes_patch.go
    scanner.go
    stream.go
    tables.go
    tags.go
)

GO_TEST_SRCS(
    bench_test.go
    decode_test.go
    encode_test.go
    fold_test.go
    fuzz_test.go
    kubernetes_patch_test.go
    number_test.go
    scanner_test.go
    stream_test.go
    tagkey_test.go
    tags_test.go
)

GO_XTEST_SRCS(
    example_marshaling_test.go
    example_test.go
    example_text_marshaling_test.go
)

END()

RECURSE(
    gotest
)
