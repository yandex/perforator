GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    httpfilter.go
)

END()

RECURSE(
    fault
    rbac
    router
)
