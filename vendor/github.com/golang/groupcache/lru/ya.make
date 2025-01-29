GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20210331224755-41bb18bfe9da)

SRCS(
    lru.go
)

GO_TEST_SRCS(lru_test.go)

END()

RECURSE(
    gotest
)
