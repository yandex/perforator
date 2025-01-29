GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.27.0)

SRCS(
    pool.go
)

GO_XTEST_SRCS(pool_test.go)

END()

RECURSE(
    gotest
)
