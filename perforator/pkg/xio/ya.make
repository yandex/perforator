GO_LIBRARY()

SRCS(
    sized.go
)

GO_TEST_SRCS(
    sized_test.go
)

END()

RECURSE(
    gotest
)
