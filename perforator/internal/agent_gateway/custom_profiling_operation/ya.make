GO_LIBRARY()

SRCS(
    config.go
    long_polling.go
    service.go
    snapshot.go
)

GO_TEST_SRCS(service_test.go)

END()

RECURSE(
    gotest
)
