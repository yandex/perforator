GO_LIBRARY()

SRCS(
    collector.go
    gc.go
    shard.go
)

GO_TEST_SRCS(
    gc_test.go
)

END()

RECURSE(gotest)
