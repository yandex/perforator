GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.8.7)

SRCS(
    decode.go
    doc.go
    integrate.go
    lex.go
    load.go
    parser.go
    properties.go
    rangecheck.go
)

GO_TEST_SRCS(
    benchmark_test.go
    decode_test.go
    example_test.go
    integrate_test.go
    load_test.go
    properties_go1.15_test.go
    properties_test.go
)

END()

RECURSE(
    assert
    gotest
)
