GO_LIBRARY()

SRCS(
    combine.go
    deduct.go
    filter.go
    models.go
    puller.go
)

GO_TEST_SRCS(
    deduct_test.go
    filter_test.go
)

END()

RECURSE(
    gotest
)
