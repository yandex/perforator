GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.1.12)

SRCS(
    binary_as_string_codec.go
    fuzzy_decoder.go
    naming_strategy.go
    privat_fields.go
    time_as_int64_codec.go
)

GO_TEST_SRCS(
    binary_as_string_codec_test.go
    fuzzy_decoder_test.go
    naming_strategy_test.go
    private_fields_test.go
    time_as_int64_codec_test.go
)

END()

RECURSE(
    gotest
)
