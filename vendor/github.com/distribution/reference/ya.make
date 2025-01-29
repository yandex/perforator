GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.5.0)

SRCS(
    helpers.go
    normalize.go
    reference.go
    regexp.go
    sort.go
)

GO_TEST_SRCS(
    fuzz_test.go
    normalize_test.go
    reference_test.go
    regexp_bench_test.go
    regexp_test.go
    sort_test.go
)

END()

RECURSE(
    gotest
)
