GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    endpoint_errors.go
    resource_request.go
)

END()

RECURSE(
    arn
    s3err
)
