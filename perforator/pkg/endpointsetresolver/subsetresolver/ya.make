GO_LIBRARY()

SRCS(
    subsetresolver.go
)

GO_TEST_SRCS(
    subsetresolver_test.go
)

END()

RECURSE(gotest)
