GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    constraints.go
    decode.go
    doc.go
    jsonschema.go
    ref.go
)

GO_TEST_SRCS(decode_test.go)

END()

RECURSE(
    gotest
)
