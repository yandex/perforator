GO_LIBRARY()

SRCS(
    postprocess.go
)

GO_TEST_SRCS(postprocess_test.go)

END()

RECURSE(
    gotest
)
