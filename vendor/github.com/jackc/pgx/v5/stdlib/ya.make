GO_LIBRARY()

LICENSE(MIT)

VERSION(v5.7.1)

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
