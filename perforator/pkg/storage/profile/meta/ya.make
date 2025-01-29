GO_LIBRARY()

SRCS(
    storage.go
)

END()

RECURSE(
    clickhouse
    config
    multi
)
