GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    common.go
)

END()

RECURSE(
    authinfo
    conn
    handshaker
    proto
    testutil
    # yo
)
