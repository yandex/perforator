GO_LIBRARY()

SRCS(
    binary_detector.go
    controller.go
    execution.go
    handler.go
    registry.go
    service.go
)

GO_TEST_SRCS(
    binary_detector_test.go
    execution_test.go
)

END()

RECURSE(
    gotest
    mocks
    models
)
