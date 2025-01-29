GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.60.0)

SRCS(
    version.go
)

END()

RECURSE(
    benchwrapper
    testutil
)
