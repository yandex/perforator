GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.42.0)

SRCS(
    codes.go
    doc.go
)

GO_TEST_SRCS(codes_test.go)

END()

RECURSE(
    gotest
)
