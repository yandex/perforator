GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20250403155104-27863c87afa6)

SRCS(
    measurement.go
)

GO_TEST_SRCS(measurement_test.go)

END()

RECURSE(
    gotest
)
