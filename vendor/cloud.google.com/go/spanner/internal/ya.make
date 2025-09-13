GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.76.1)

SRCS(
    version.go
)

END()

RECURSE(
    benchwrapper
    testutil
)
