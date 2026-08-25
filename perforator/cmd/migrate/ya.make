GO_PROGRAM()

SRCS(
    logger.go
    main.go
)

GO_TEST_SRCS(
    main_test.go
)

GO_EMBED_PATTERN(migrations/clickhouse/*.sql)

GO_EMBED_PATTERN(migrations/postgres/*.sql)

END()

RECURSE(
    gotest
)
