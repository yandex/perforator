GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    map.go
    name.go
    sort.go
    sort_values.go
    string.go
    t.go
)

GO_TEST_SRCS(
    map_private_test.go
    sort_test.go
)

GO_XTEST_SRCS(
    map_test.go
    name_test.go
    sort_values_test.go
    string_test.go
    # t_test.go
)

END()

RECURSE(
    gotest
)
