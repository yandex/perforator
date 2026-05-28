GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.1.5)

SRCS(
    transfer.go
)

END()

RECURSE(
    proxy
    streaming
)
