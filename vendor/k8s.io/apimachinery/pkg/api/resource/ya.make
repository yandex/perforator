GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    amount.go
    generated.pb.go
    math.go
    quantity.go
    quantity_proto.go
    scale_int.go
    suffix.go
    zz_generated.deepcopy.go
)

GO_TEST_SRCS(
    amount_test.go
    math_test.go
    quantity_proto_test.go
    quantity_test.go
    scale_int_test.go
)

GO_XTEST_SRCS(quantity_example_test.go)

END()

RECURSE(
    gotest
)
