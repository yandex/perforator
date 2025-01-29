GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.19.1)

SRCS(
    doc.go
    parse_req.go
    supported_features.go
)

GO_XTEST_SRCS(parse_req_test.go)

END()

RECURSE(
    gotest
)
