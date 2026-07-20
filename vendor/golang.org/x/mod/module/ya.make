GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.37.0)

SRCS(
    module.go
    pseudo.go
)

GO_TEST_SRCS(
    module_test.go
    pseudo_test.go
)

END()

RECURSE(
    gotest
)
