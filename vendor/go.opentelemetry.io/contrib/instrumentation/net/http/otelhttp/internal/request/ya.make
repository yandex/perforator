GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.57.0)

SRCS(
    body_wrapper.go
    resp_writer_wrapper.go
)

GO_TEST_SRCS(
    body_wrapper_test.go
    resp_writer_wrapper_test.go
)

END()

RECURSE(
    gotest
)
