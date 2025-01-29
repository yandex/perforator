GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    detect.go
    encoder.go
    encoding.go
)

GO_TEST_SRCS(
    detect_test.go
    encoding_test.go
)

END()

RECURSE(
    gotest
    json
    yaml
)
