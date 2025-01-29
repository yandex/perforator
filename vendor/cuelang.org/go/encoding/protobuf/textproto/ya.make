GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    decoder.go
    doc.go
    encoder.go
)

GO_XTEST_SRCS(
    # decoder_test.go
    encoder_test.go
)

END()

RECURSE(
    gotest
)
