GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.5.4)

SRCS(
    case.go
    delete.go
    delete_ctx.go
    expr.go
    insert.go
    insert_ctx.go
    part.go
    placeholder.go
    row.go
    select.go
    select_ctx.go
    squirrel.go
    squirrel_ctx.go
    statement.go
    stmtcacher.go
    stmtcacher_ctx.go
    update.go
    update_ctx.go
    where.go
)

GO_TEST_SRCS(
    case_test.go
    delete_ctx_test.go
    delete_test.go
    expr_test.go
    insert_ctx_test.go
    insert_test.go
    placeholder_test.go
    row_test.go
    select_ctx_test.go
    select_test.go
    squirrel_ctx_test.go
    squirrel_test.go
    statement_test.go
    stmtcacher_ctx_test.go
    stmtcacher_test.go
    update_ctx_test.go
    update_test.go
    where_test.go
)

END()

RECURSE(
    gotest
)
