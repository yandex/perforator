GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20250923004556-9e5a51aed1e8)

SRCS(
    measurement.go
)

GO_TEST_SRCS(measurement_test.go)

END()

RECURSE(
    gotest
)
