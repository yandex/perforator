GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v0.0.0-20241010143419-9aa6b5e7a4b3)

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
