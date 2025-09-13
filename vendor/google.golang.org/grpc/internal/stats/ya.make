GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    labels.go
    metrics_recorder_list.go
)

GO_XTEST_SRCS(metrics_recorder_list_test.go)

END()

RECURSE(
    gotest
)
