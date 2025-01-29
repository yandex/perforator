GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    builtin.go
    context.go
    errors.go
    register.go
)

END()

RECURSE(
    builtintest
)
