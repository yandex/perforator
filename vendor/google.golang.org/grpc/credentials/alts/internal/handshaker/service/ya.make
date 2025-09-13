GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    service.go
)

GO_TEST_SRCS(service_test.go)

END()

RECURSE(
    gotest
)
