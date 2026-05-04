GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v0.63.0)

SRCS(
    body_wrapper.go
    gen.go
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
