GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.12.2)

SRCS(
    conn_str.go
    conn_str_go115.go
    conn_str_go118.go
)

GO_TEST_SRCS(conn_str_test.go)

END()

RECURSE(
    gotest
)
