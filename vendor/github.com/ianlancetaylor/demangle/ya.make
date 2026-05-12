GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.0.0-20250417193237-f615e6bd150b)

SRCS(
    ast.go
    demangle.go
    rust.go
)

GO_TEST_SRCS(
    ast_test.go
    cases_test.go
    demangle_test.go
    expected_test.go
    rust_expected_test.go
)

END()

RECURSE(
    gotest
)
