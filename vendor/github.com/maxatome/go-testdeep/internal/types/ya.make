GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    order.go
    reflect.go
    types.go
)

GO_TEST_SRCS(types_private_test.go)

GO_XTEST_SRCS(
    order_test.go
    reflect_go120_test.go
    reflect_test.go
    types_test.go
)

END()

RECURSE(
    gotest
)
