GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.37.0)

SRCS(
    doc.go
    event.go
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
