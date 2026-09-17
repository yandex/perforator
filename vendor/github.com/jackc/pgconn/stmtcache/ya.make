GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.14.3)

SRCS(
    lru.go
    stmtcache.go
)

GO_XTEST_SRCS(
    # lru_test.go
)

END()

RECURSE(
    gotest
)
