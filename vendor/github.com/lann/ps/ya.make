GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.0.0-20150810152359-62de8c46ede0)

SRCS(
    list.go
    map.go
)

GO_TEST_SRCS(
    list_test.go
    map_test.go
)

END()

RECURSE(
    gotest
)
