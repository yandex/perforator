GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20260721132016-d427ff9ee9ad)

SRCS(
    doc.go
    document.go
    document_v3.go
    openapi.go
)

END()

RECURSE(
    testing
    validation
)
