GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.42.0)

SRCS(
    doc.go
    exception.go
    http.go
    resource.go
    schema.go
    trace.go
)

END()

RECURSE(
    httpconv
    netconv
)
