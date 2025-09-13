GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    map.go
    resolver.go
)

GO_TEST_SRCS(
    map_test.go
    resolver_test.go
)

END()

RECURSE(
    dns
    gotest
    manual
    passthrough
    ringhash
    # yo
)
