GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.37.0)

SRCS(
    handler.go
    instruments.go
    internal_logging.go
    meter.go
    propagator.go
    state.go
    trace.go
)

GO_TEST_SRCS(
    alternate_meter_test.go
    benchmark_test.go
    handler_test.go
    instruments_test.go
    internal_logging_test.go
    meter_test.go
    meter_types_test.go
    propagator_test.go
    state_test.go
    trace_test.go
    util_test.go
)

END()

RECURSE(
    gotest
)
