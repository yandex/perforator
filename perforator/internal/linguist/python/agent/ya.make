GO_LIBRARY()

SRCS(
    offsets_registry.go
    registry.go
    stackprocessor.go
    symbolizer.go
)

GO_TEST_SRCS(
    offsets_registry_test.go
    registry_test.go
    stackprocessor_test.go
    symbolizer_test.go
)

END()

RECURSE(
    linetable
    remotemem
)

RECURSE_FOR_TESTS(
    gotest
)
