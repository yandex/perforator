GO_LIBRARY()

SRCS(
    counter.go
    gauge.go
    histogram.go
    int_gauge.go
    registry.go
    registry_getters.go
    registry_opts.go
    timer.go
    vec.go
)

GO_TEST_SRCS(registry_test.go)

END()

RECURSE(
    gotest
)
