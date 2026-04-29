GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.42.0)

SRCS(
    doc.go
    encoder.go
    filter.go
    hash.go
    iterator.go
    key.go
    kv.go
    rawhelpers.go
    set.go
    type_string.go
    value.go
)

GO_TEST_SRCS(
    filter_test.go
    hash_test.go
)

GO_XTEST_SRCS(
    benchmark_test.go
    iterator_test.go
    key_test.go
    kv_test.go
    set_test.go
    value_test.go
)

END()

RECURSE(
    gotest
    internal
)
