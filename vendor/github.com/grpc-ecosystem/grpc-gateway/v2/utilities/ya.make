GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.26.3)

SRCS(
    doc.go
    pattern.go
    readerfactory.go
    string_array_flag.go
    trie.go
)

GO_XTEST_SRCS(
    string_array_flag_test.go
    trie_test.go
)

END()

RECURSE(
    gotest
)
