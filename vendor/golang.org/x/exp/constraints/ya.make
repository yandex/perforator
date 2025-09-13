GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.0.0-20250305212735-054e65f0b394)

SRCS(
    constraints.go
)

GO_TEST_SRCS(constraints_test.go)

END()

RECURSE(
    gotest
)
