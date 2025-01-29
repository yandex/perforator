GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.5.2)

SRCS(
    argument.go
    column.go
    driver.go
    expectations.go
    expectations_go18.go
    options.go
    query.go
    result.go
    rows.go
    rows_go18.go
    sqlmock.go
    sqlmock_go18.go
    sqlmock_go19.go
    statement.go
    statement_go18.go
)

GO_TEST_SRCS(
    argument_test.go
    column_test.go
    driver_test.go
    expectations_go18_test.go
    expectations_go19_test.go
    expectations_test.go
    query_test.go
    result_test.go
    rows_go13_test.go
    rows_go18_test.go
    rows_test.go
    sqlmock_go18_test.go
    sqlmock_go19_test.go
    sqlmock_test.go
    statement_test.go
    stubs_test.go
)

END()

RECURSE(
    examples
    gotest
)
