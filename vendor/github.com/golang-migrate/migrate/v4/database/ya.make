GO_LIBRARY()

LICENSE(MIT)

VERSION(v4.15.2)

SRCS(
    driver.go
    error.go
    util.go
)

END()

RECURSE(
    cassandra
    clickhouse
    cockroachdb
    mongodb
    multistmt
    mysql
    pgx
    postgres
    redshift
    spanner
    sqlite
    sqlite3
    sqlserver
    stub
    testing
)
