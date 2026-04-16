GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

SRCS(
    httpfilter.go
)

END()

RECURSE(
    fault
    rbac
    router
)
