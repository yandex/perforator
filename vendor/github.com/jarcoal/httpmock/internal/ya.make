GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.3.1)

SRCS(
    error.go
    route_key.go
    stack_tracer.go
    submatches.go
)

GO_XTEST_SRCS(
    error_test.go
    route_key_test.go
    stack_tracer_test.go
    submatches_test.go
)

END()

RECURSE(
    gotest
)
