GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    adt.go
    binop.go
    closed.go
    closed2.go
    composite.go
    comprehension.go
    context.go
    decimal.go
    default.go
    disjunct.go
    doc.go
    equality.go
    errors.go
    eval.go
    expr.go
    feature.go
    kind.go
    op.go
    optional.go
    simplify.go
)

GO_TEST_SRCS(
    expr_test.go
    kind_test.go
)

GO_XTEST_SRCS(
    # closed_test.go
    # eval_test.go
    # feature_test.go
    # optional_test.go
)

END()

RECURSE(
    gotest
)
