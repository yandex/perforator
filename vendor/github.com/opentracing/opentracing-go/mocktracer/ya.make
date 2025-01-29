GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.2.0)

SRCS(
    mocklogrecord.go
    mockspan.go
    mocktracer.go
    propagation.go
)

GO_TEST_SRCS(mocktracer_test.go)

END()

RECURSE(
    gotest
)
