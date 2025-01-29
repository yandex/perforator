GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    manager.go
    producer.go
)

GO_TEST_SRCS(manager_test.go)

END()

RECURSE(
    gotest
)
