GO_LIBRARY()

SRCS(
    polling.go
    resolver.go
    source.go
)

GO_TEST_SRCS(
    polling_test.go
    resolver_test.go
)

END()

RECURSE_FOR_TESTS(gotest)

RECURSE(
    kubelet
)
