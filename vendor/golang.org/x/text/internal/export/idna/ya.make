GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.38.0)

SRCS(
    idna.go
    punycode.go
    tables15.0.0.go
    trie.go
    trieval.go
)

GO_TEST_SRCS(
    common_test.go
    conformance_test.go
    gen_test.go
    idna_test.go
    punycode_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
