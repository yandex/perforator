GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

SRCS(
    config.go
    transport_builder.go
)

END()

RECURSE(
    grpctransport
    internal
    lrsclient
    xdsclient
)
