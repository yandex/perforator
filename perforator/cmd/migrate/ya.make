GO_PROGRAM()

SRCS(
    logger.go
    main.go
)

GO_EMBED_PATTERN(migrations/clickhouse/*.sql)

GO_EMBED_PATTERN(migrations/postgres/*.sql)

END()
