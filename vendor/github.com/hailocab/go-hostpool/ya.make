GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.0.0-20160125115350-e80d13ce29ed)

SRCS(
    epsilon_greedy.go
    epsilon_value_calculators.go
    host_entry.go
    hostpool.go
)

GO_TEST_SRCS(
    # example_test.go
    hostpool_test.go
)

END()

RECURSE(
    gotest
)
