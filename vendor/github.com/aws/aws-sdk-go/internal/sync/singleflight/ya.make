GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    singleflight.go
)

GO_TEST_SRCS(singleflight_test.go)

END()

RECURSE(
    gotest
)
