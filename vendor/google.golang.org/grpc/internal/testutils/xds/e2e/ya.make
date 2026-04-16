GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

SRCS(
    bootstrap.go
    clientresources.go
    logging.go
    server.go
)

END()

RECURSE(
    setup
)
