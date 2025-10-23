GO_LIBRARY()

SRCS(
    agent.go
    config.go
    debug_toggler.go
)

END()

RECURSE(
    custom_profiling_operation
)
