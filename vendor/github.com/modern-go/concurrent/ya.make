GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20180306012644-bacd9c7ef1dd)

SRCS(
    executor.go
    go_above_19.go
    log.go
    unbounded_executor.go
)

GO_XTEST_SRCS(
    map_test.go
    unbounded_executor_test.go
)

END()

RECURSE(
    gotest
)
