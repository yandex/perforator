GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.6)

SRCS(
    types.go
)

END()

RECURSE(
    awserr
    credentials
    signer
)
