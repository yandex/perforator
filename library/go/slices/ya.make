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
    reverse.go
    shuffle.go
    sort.go
    subtract.go
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
    reverse_test.go
    shuffle_test.go
    subtract_test.go
)

END()

RECURSE(
    gotest
)
