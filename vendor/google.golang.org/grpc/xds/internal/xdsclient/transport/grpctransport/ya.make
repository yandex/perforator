GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    grpctransport.go
)

GO_XTEST_SRCS(grpctransport_ext_test.go)

END()

RECURSE(
    gotest
)
