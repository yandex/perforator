GO_LIBRARY()

LICENSE(MIT)

VERSION(v5.7.1)

SRCS(
    bgreader.go
)

GO_XTEST_SRCS(bgreader_test.go)

END()

RECURSE(
    gotest
)
