GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.23.2)

SRCS(
    problem.go
    promlint.go
    validation.go
)

GO_XTEST_SRCS(promlint_test.go)

END()

RECURSE(
    gotest
    validations
)
