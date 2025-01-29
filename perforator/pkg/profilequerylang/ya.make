GO_LIBRARY()

SRCS(
    builder.go
    labels.go
    parse.go
    selector.go
)

GO_TEST_SRCS(selector_test.go)

END()

RECURSE(
    gotest
)
