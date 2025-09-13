GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    observability.go
    pluginoption.go
)

GO_TEST_SRCS(
    observability_test.go
    pluginoption_test.go
)

END()

RECURSE(
    gotest
)
