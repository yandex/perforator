GO_LIBRARY()

SRCS(
    chunk.go
    contains.go
    dedup.go
    equal.go
    filter.go
    group_by.go
    intersects.go
    join.go
    map.go
    map_async.go
    merge_sorted.go
    reverse.go
    shuffle.go
    sort.go
    subtract.go
    union.go
    zip.go
)

GO_XTEST_SRCS(
    chunk_test.go
    dedup_test.go
    equal_test.go
    filter_test.go
    group_by_test.go
    intersects_test.go
    join_test.go
    map_test.go
    map_async_test.go
    merge_sorted_test.go
    reverse_test.go
    shuffle_test.go
    subtract_test.go
    union_test.go
    zip_test.go
)

END()

RECURSE(
    gotest
)
