GO_LIBRARY()

SRCS(
    config.go
    server.go
)

END()

RECURSE(
    custom_profiling_operation
    storage
)
