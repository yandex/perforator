GO_LIBRARY()

LICENSE(MIT)

VERSION(v4.18.3)

SRCS(
    sql.go
)

GO_XTEST_SRCS(
    # bench_test.go
    # sql_test.go
)

END()

RECURSE(
    gotest
)
