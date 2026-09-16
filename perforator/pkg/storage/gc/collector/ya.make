GO_LIBRARY()

SRCS(
    metrics.go
    gc.go
    storage_gc.go
    cluster_top_gc.go
)

GO_TEST_SRCS(
    gc_test.go
    cluster_top_gc_test.go
    lease_test.go
    run_test.go
)

END()

RECURSE(gotest)
