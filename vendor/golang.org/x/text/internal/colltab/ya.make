GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.21.0)

SRCS(
    collelem.go
    colltab.go
    contract.go
    iter.go
    numeric.go
    table.go
    trie.go
    weighter.go
)

GO_TEST_SRCS(
    collelem_test.go
    colltab_test.go
    contract_test.go
    iter_test.go
    numeric_test.go
    trie_test.go
    weighter_test.go
)

GO_XTEST_SRCS(collate_test.go)

END()

RECURSE(
    # gotest
)
