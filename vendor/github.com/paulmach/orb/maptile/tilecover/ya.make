GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.11.1)

SRCS(
    helpers.go
    line_string.go
    merge.go
    polygon.go
)

GO_TEST_SRCS(
    benchmarks_test.go
    cover_test.go
    helpers_test.go
    merge_test.go
    polygon_test.go
)

END()

RECURSE(
    gotest
)
