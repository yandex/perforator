GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

ALL_GO_TEST_SRCS()

SRCS(
    extconfig.go
    httpfilter.go
)

END()

RECURSE(
    extproc
    fault
    gcp_authn
    gotest
    rbac
    router
)
