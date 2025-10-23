GO_LIBRARY()

SRCS(
    controller.go
    execution.go
    handler.go
    registry.go
    service.go
)

GO_TEST_SRCS(execution_test.go)

END()

RECURSE(
    gotest
    mocks
    models
)
