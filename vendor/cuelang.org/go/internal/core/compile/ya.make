GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

#GO_XTEST_SRCS(compile_test.go)

SRCS(
    builtin.go
    compile.go
    errors.go
    label.go
    predeclared.go
)

GO_XTEST_SRCS(compile_test.go)

END()

RECURSE(
    gotest
)
