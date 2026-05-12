GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.4)

SRCS(
    readconcern.go
)

GO_XTEST_SRCS(readconcern_test.go)

END()

RECURSE(
    gotest
)
