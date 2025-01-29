GO_TEST()

LICENSE(MIT)

VERSION(v1.1.12)

GO_TEST_SRCS(
    config_test.go
    decoder_test.go
    # encoder_18_test.go
    encoder_test.go
    marshal_indent_test.go
    marshal_json_escape_test.go
    marshal_json_test.go
)

END()
