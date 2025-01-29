GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    service.go
)

GO_TEST_SRCS(service_test.go)

END()

RECURSE(
    gotest
)
