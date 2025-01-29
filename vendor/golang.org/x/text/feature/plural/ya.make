GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.21.0)

SRCS(
    common.go
    message.go
    plural.go
    tables.go
)

GO_TEST_SRCS(
    data_test.go
    message_test.go
    plural_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    # gotest
)
