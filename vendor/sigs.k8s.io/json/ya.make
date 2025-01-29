GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v0.0.0-20221116044647-bc3834ca7abd)

SRCS(
    doc.go
    json.go
)

GO_TEST_SRCS(json_test.go)

END()

RECURSE(
    gotest
    internal
)
