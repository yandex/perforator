GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.0.8)

SRCS(
    route_key.go
    stack_tracer.go
    submatches.go
)

GO_XTEST_SRCS(
    route_key_test.go
    stack_tracer_test.go
    submatches_test.go
)

END()

RECURSE(
    gotest
)
