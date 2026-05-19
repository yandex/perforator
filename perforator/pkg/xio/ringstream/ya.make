GO_LIBRARY()

SRCS(
    adapter.go
    interval_set.go
    ring_buf.go
)

GO_TEST_SRCS(
    adapter_test.go
    interval_set_test.go
    ring_buf_test.go
)

END()

RECURSE(
    gotest
)
