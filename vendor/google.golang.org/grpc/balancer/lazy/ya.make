GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

SRCS(
    lazy.go
)

GO_XTEST_SRCS(lazy_ext_test.go)

END()

RECURSE(
    gotest
)
