GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

SRCS(
    labels.go
    metrics_recorder_list.go
    stats.go
)

GO_XTEST_SRCS(
    labels_test.go
    metrics_recorder_list_test.go
)

END()

RECURSE(
    gotest
)
